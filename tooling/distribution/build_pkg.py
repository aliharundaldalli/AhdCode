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
import shutil
import subprocess
import tempfile
from xml.etree import ElementTree

ROOT = pathlib.Path(__file__).resolve().parents[2]
IDENTIFIER = 'com.ahdcode.toolchain'
INSTALL_ROOT = 'Library/AhdCode'

DISTRIBUTION = '''<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
    <title>AhdCode {version}</title>
    <organization>com.ahdcode</organization>
    <options customize="never" require-scripts="true" hostArchitectures="arm64"/>
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

WELCOME = '''AhdCode {version}

This installs the AhdCode compiler, its private Go toolchain, AhdDataStudio,
the SQLite, numeric and plot helpers, an offline LaTeX engine, the project
starters and the English documentation.

Everything is installed for your account only, in your home folder:

    ~/Library/AhdCode

No administrator password is required. Nothing is placed in system folders.
Your existing AhdCode projects, databases and settings are not touched.
'''

CONCLUSION = '''AhdCode {version} is installed.

Open a new Terminal window and run:

    ahdcode --version

The installer added AhdCode to your PATH. A terminal that was already open
keeps the environment it started with, so open a new one.

The Visual Studio Code extension is in:

    ~/Library/AhdCode/current/vscode

Install it from VS Code with Extensions, the "..." menu, Install from VSIX.
'''


def run(arguments):
    subprocess.run([str(a) for a in arguments], check=True)



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


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--payload', type=pathlib.Path, required=True,
                        help="the staged payload directory (stage-macos-arm64/payload)")
    parser.add_argument('--version', required=True)
    parser.add_argument('--output', type=pathlib.Path, required=True)
    parser.add_argument('--sign', default='',
                        help='Developer ID Installer identity; unsigned when omitted')
    arguments = parser.parse_args()

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
        (resources / 'welcome.txt').write_text(WELCOME.format(version=arguments.version))
        (resources / 'conclusion.txt').write_text(CONCLUSION.format(version=arguments.version))
        (work / 'distribution.xml').write_text(DISTRIBUTION.format(
            version=arguments.version, identifier=IDENTIFIER, component=component.name))

        product = ['productbuild', '--distribution', work / 'distribution.xml',
                   '--package-path', work, '--resources', resources]
        if arguments.sign:
            product += ['--sign', arguments.sign]
        product.append(arguments.output)
        run(product)

    verify(arguments.output, arguments.version, bool(arguments.sign))
    print('Built', arguments.output, arguments.output.stat().st_size, 'bytes')


def verify(package, version, signed):
    """Check the shape a Mac actually installs, not merely that a file exists."""
    with tempfile.TemporaryDirectory(prefix='ahdcode-pkg-check-') as workspace:
        expanded = pathlib.Path(workspace) / 'expanded'
        run(['pkgutil', '--expand', package, expanded])
        distribution = (expanded / 'Distribution').read_text()
        for required in ['enable_currentUserHome="true"', 'enable_localSystem="false"', IDENTIFIER]:
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
        if not (component / 'Scripts').exists():
            raise SystemExit('the package carries no postinstall script')
        authorization = ElementTree.parse(descriptor).getroot().get('auth')
        if authorization != 'none':
            raise SystemExit('the package would ask for an administrator password (auth=%s)' % authorization)
    result = subprocess.run(['pkgutil', '--check-signature', str(package)],
                            capture_output=True, text=True)
    status = 'signed' if 'Status: signed' in result.stdout else 'unsigned'
    if signed and status != 'signed':
        raise SystemExit('signing was requested but the package is not signed')
    print('Package verification: install root', INSTALL_ROOT + ', home-directory domain, no administrator password,', status)


if __name__ == '__main__':
    main()
