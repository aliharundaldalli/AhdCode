#!/usr/bin/env python3
"""Build the macOS installer package from an already-staged payload.

A .pkg is what a Mac user expects: double-click, Continue, Install, Done. It
installs into the user's own home, so no administrator password is asked for,
and it never opens a terminal.

The installation root is ~/Library/AhdCode. It deliberately avoids
~/Library/Application Support/AhdCode, which is where AhdCode keeps the user's
database registry and local routes -- on a case-insensitive volume that is the
same directory as the toolchain's own ahdcode data folder, so installing there
either refused to run or would have put user data inside something an uninstall
deletes.
"""
import argparse
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
from xml.etree import ElementTree

ROOT = pathlib.Path(__file__).resolve().parents[2]
IDENTIFIER = 'com.ahdcode.toolchain'
INSTALL_ROOT = 'Library/AhdCode'
HOST_ARCHITECTURES = {
    'arm64': 'arm64',
    'amd64': 'x86_64',
}

DISTRIBUTION = '''<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
    <title>AhdCode {version}</title>
    <organization>com.ahdcode</organization>
    <options customize="never" require-scripts="true" hostArchitectures="{host_arch}"/>
    <!-- Home directory only: no administrator password, nothing outside the
         installing user's account. -->
    <domains enable_anywhere="false" enable_currentUserHome="true" enable_localSystem="false"/>
    <welcome file="welcome.txt" mime-type="text/plain"/>
    <conclusion file="conclusion.txt" mime-type="text/plain"/>
    <choices-outline><line choice="default"/></choices-outline>
    <choice id="default" title="AhdCode"><pkg-ref id="{identifier}"/></choice>
    <pkg-ref id="{identifier}" version="{version}">{component}</pkg-ref>
</installer-gui-script>
'''


def build_welcome(version, arch, payload, signed, notarized):
    components = [
        'the AhdCode compiler (ahdcode)',
        'SQLite, numeric, plot, graphics and GUI helpers',
        'embedded AhdDataStudio, starter templates and English documentation',
    ]
    if (payload / 'libexec' / 'go').is_dir():
        components.append('the bundled private Go toolchain')
    if (payload / 'libexec' / 'ahdcode' / 'latex').is_dir():
        components.append('the offline Tectonic LaTeX engine')
    if (payload / 'vscode').is_dir():
        components.append('the Visual Studio Code extension (.vsix)')

    if signed and notarized:
        trust_note = 'This package is signed with an Apple Developer ID and notarized by Apple.'
    elif signed:
        trust_note = 'NOTICE: This package is signed but not notarized by Apple.'
    else:
        trust_note = (
            'NOTICE: This is an UNSIGNED development / release-candidate package.\n'
            'Verify its SHA-256 checksum before installing, and never disable Gatekeeper.'
        )

    return (
        f'AhdCode {version} (macOS {arch})\n\n'
        'This installs:\n'
        + ''.join(f'  - {item}\n' for item in components)
        + '\n'
        f'Files go into a versioned directory under ~/Library/AhdCode/versions/{version}.\n'
        'Previously installed versions (including 2.9.0) are preserved intact, and\n'
        '~/Library/AhdCode/current is updated to point to this version.\n\n'
        'No administrator password is required. Nothing is written to system folders.\n'
        'Your existing AhdCode projects, databases and settings are not touched.\n\n'
        f'{trust_note}\n'
    )


def build_conclusion(version, arch, payload):
    lines = [
        f'AhdCode {version} (macOS {arch}) is installed.',
        '',
        'Open a new Terminal window and run:',
        '',
        '    ahdcode version',
        '',
        'The installer added ~/Library/AhdCode/bin to your shell profile PATH.',
        'Any terminal that was already open keeps its previous environment.',
        '',
    ]
    if not (payload / 'libexec' / 'go').is_dir():
        lines += [
            'This toolchain package relies on the Go version from go.mod in PATH',
            'for native `ahdcode build` and `ahdcode package` commands.',
            '',
        ]
    if (payload / 'vscode').is_dir():
        lines += [
            'The Visual Studio Code extension is in:',
            '',
            '    ~/Library/AhdCode/current/vscode',
            '',
            'Install it from VS Code with Extensions, "...", Install from VSIX.',
            '',
        ]
    lines += [
        'To roll back to a previously installed version (for example 2.9.0):',
        '',
        '    ln -sfn versions/2.9.0 ~/Library/AhdCode/.current-new && mv -fh ~/Library/AhdCode/.current-new ~/Library/AhdCode/current',
        '',
        'To uninstall AhdCode and remove its managed PATH block:',
        '',
        '    sh "$HOME/Library/AhdCode/current/install.sh" --uninstall',
        '',
    ]
    return '\n'.join(lines)


def run(arguments):
    subprocess.run([str(a) for a in arguments], check=True)


def validate_version(version):
    if not version or '..' in version or not re.fullmatch(r'[0-9A-Za-z._-]+', version):
        raise SystemExit('invalid version string for macOS package: ' + repr(version))


def relax_authorization(component, workspace):
    """pkgbuild marks a component auth="root", which makes the installer ask for
    an administrator password. This package installs only into the user's own
    home, so it needs no elevation at all; the attribute is rewritten and the
    component re-flattened."""
    run(['pkgutil', '--expand', component, workspace])
    descriptor = workspace / 'PackageInfo'
    text = descriptor.read_text()
    updated = text.replace('auth="root"', 'auth="none"', 1)
    if updated == text and 'auth="none"' not in text:
        raise SystemExit('could not find the authorization attribute in PackageInfo')
    descriptor.write_text(updated)
    component.unlink()
    run(['pkgutil', '--flatten', workspace, component])


def notarize_and_staple(package, profile):
    run(['xcrun', 'notarytool', 'submit', str(package), '--keychain-profile', profile, '--wait'])
    run(['xcrun', 'stapler', 'staple', str(package)])
    run(['xcrun', 'stapler', 'validate', str(package)])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--payload', type=pathlib.Path, required=True,
                        help='the staged payload directory (stage-macos-arm64/payload)')
    parser.add_argument('--version', required=True)
    parser.add_argument('--arch', choices=sorted(HOST_ARCHITECTURES), default='arm64',
                        help='target macOS architecture (arm64 or amd64)')
    parser.add_argument('--output', type=pathlib.Path, required=True)
    parser.add_argument('--sign', default='',
                        help='Developer ID Installer identity; unsigned when omitted')
    parser.add_argument('--notary-profile', default='',
                        help='xcrun notarytool keychain profile name for notarizing and stapling signed packages')
    parser.add_argument('--leak-forbid-file', type=pathlib.Path, action='append', default=[],
                        help='a developer-local file whose exact contents must not appear in the package')
    parser.add_argument('--leak-forbid-env-values', type=pathlib.Path, action='append', default=[],
                        help='a developer .env file whose specific values must not appear in the package')
    arguments = parser.parse_args()

    validate_version(arguments.version)
    host_arch = HOST_ARCHITECTURES[arguments.arch]
    if arguments.notary_profile and not arguments.sign:
        raise SystemExit('--notary-profile requires --sign')

    payload = arguments.payload.resolve()
    if not (payload / 'bin' / 'ahdcode').is_file():
        raise SystemExit('payload does not contain bin/ahdcode: ' + str(payload))
    arguments.output.parent.mkdir(parents=True, exist_ok=True)

    with tempfile.TemporaryDirectory(prefix='ahdcode-pkg-') as workspace:
        work = pathlib.Path(workspace)
        # The payload is laid out exactly as it must appear under the install
        # root, so pkgbuild copies it verbatim.
        staged = work / 'root' / 'versions' / arguments.version
        staged.parent.mkdir(parents=True)
        shutil.copytree(payload, staged, symlinks=True)
        if not (staged / 'install.sh').is_file():
            shutil.copy2(ROOT / 'tooling/distribution/install.sh', staged / 'install.sh')
        (staged / 'install.sh').chmod(0o755)
        (staged / 'VERSION').write_text(arguments.version + '\n')
        (work / 'root' / 'versions' / 'VERSION.current').write_text(arguments.version + '\n')

        scripts = work / 'scripts'
        scripts.mkdir()
        shutil.copy2(ROOT / 'tooling/distribution/macos/postinstall', scripts / 'postinstall')
        (scripts / 'postinstall').chmod(0o755)

        component = work / 'component.pkg'
        run(['pkgbuild', '--root', work / 'root', '--scripts', scripts,
             '--identifier', IDENTIFIER, '--version', arguments.version,
             '--install-location', INSTALL_ROOT, '--ownership', 'recommended', component])
        relax_authorization(component, work / 'component-expanded')

        resources = work / 'resources'
        resources.mkdir()
        (resources / 'welcome.txt').write_text(
            build_welcome(arguments.version, arguments.arch, payload, bool(arguments.sign), bool(arguments.notary_profile))
        )
        (resources / 'conclusion.txt').write_text(
            build_conclusion(arguments.version, arguments.arch, payload)
        )
        (work / 'distribution.xml').write_text(DISTRIBUTION.format(
            version=arguments.version,
            host_arch=host_arch,
            identifier=IDENTIFIER,
            component=component.name,
        ))

        product = ['productbuild', '--distribution', work / 'distribution.xml',
                   '--package-path', work, '--resources', resources]
        if arguments.sign:
            product += ['--sign', arguments.sign]
        product.append(arguments.output)
        run(product)

    # Run leak gate before optional network notarization so private files can never leave the machine.
    gate = [sys.executable, ROOT / 'tooling/distribution/leak_gate.py']
    for path in arguments.leak_forbid_file:
        gate += ['--forbid-file', path]
    for path in arguments.leak_forbid_env_values:
        gate += ['--forbid-env-values', path]
    run(gate + [arguments.output])

    if arguments.notary_profile:
        notarize_and_staple(arguments.output, arguments.notary_profile)

    verify(arguments.output, arguments.version, bool(arguments.sign), host_arch=host_arch)
    print('Built', arguments.output, arguments.output.stat().st_size, 'bytes')


def verify(package, version, signed, host_arch='arm64'):
    """Check the shape a Mac actually installs, not merely that a file exists."""
    with tempfile.TemporaryDirectory(prefix='ahdcode-pkg-check-') as workspace:
        expanded = pathlib.Path(workspace) / 'expanded'
        run(['pkgutil', '--expand-full', package, expanded])
        distribution = (expanded / 'Distribution').read_text()
        for required in [
            'enable_currentUserHome="true"',
            'enable_localSystem="false"',
            f'hostArchitectures="{host_arch}"',
            IDENTIFIER,
        ]:
            if required not in distribution:
                raise SystemExit('the package does not declare ' + required)
        component = next(expanded.glob('*.pkg'), None)
        if component is None:
            raise SystemExit('the package carries no component')
        # PackageInfo is plain XML, not a property list.
        descriptor = component / 'PackageInfo'
        if not descriptor.exists():
            raise SystemExit('the component carries no PackageInfo')
        location = ElementTree.parse(descriptor).getroot().get('install-location', '')
        if location.strip('/') != INSTALL_ROOT:
            raise SystemExit('unexpected install location: ' + location)
        if not (component / 'Scripts' / 'postinstall').is_file():
            raise SystemExit('the package carries no postinstall script')
        installed_ahd = component / 'Payload' / 'versions' / version / 'bin' / 'ahdcode'
        installed_sh = component / 'Payload' / 'versions' / version / 'install.sh'
        if not installed_ahd.is_file() or not (installed_ahd.stat().st_mode & 0o111):
            raise SystemExit('the package payload is missing executable bin/ahdcode')
        if not installed_sh.is_file() or not (installed_sh.stat().st_mode & 0o111):
            raise SystemExit('the package payload is missing executable install.sh')
        authorization = ElementTree.parse(descriptor).getroot().get('auth')
        if authorization != 'none':
            raise SystemExit('the package would ask for an administrator password (auth=%s)' % authorization)
    result = subprocess.run(['pkgutil', '--check-signature', str(package)],
                            capture_output=True, text=True)
    status = 'signed' if 'Status: signed' in result.stdout else 'unsigned'
    if signed and status != 'signed':
        raise SystemExit('signing was requested but the package is not signed')
    print(
        'Package verification: install root',
        INSTALL_ROOT + f', hostArchitectures={host_arch}, home-directory domain, no administrator password,',
        status,
    )


if __name__ == '__main__':
    main()
