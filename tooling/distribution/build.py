#!/usr/bin/env python3
"""Build self-contained release artifacts. Native builds run strictly sequentially."""
import argparse, hashlib, json, os, pathlib, re, shutil, subprocess, sys, tarfile, time, zipfile

R = pathlib.Path(__file__).resolve().parents[2]


def sha(p):
    h = hashlib.sha256()
    with open(p, 'rb') as f:
        for b in iter(lambda: f.read(1024 * 1024), b''):
            h.update(b)
    return h.hexdigest()


def run(args, **kwargs):
    subprocess.run([str(a) for a in args], check=True, **kwargs)


def extract(archive, dest):
    if str(archive).endswith('.zip'):
        with zipfile.ZipFile(archive) as z:
            z.extractall(dest)
    else:
        with tarfile.open(archive) as t:
            t.extractall(dest, filter='data')


def build(target, output, goos, arch, ldflags='-buildid=', module=None):
    env = dict(os.environ, GOFLAGS='-p=1', GOMAXPROCS='2', GOOS=goos, GOARCH=arch, CGO_ENABLED='0')
    print('Building', goos, arch, target, flush=True)
    run(['nice', '-n', '10', 'go', 'build', '-trimpath', '-ldflags=' + ldflags, '-o', output, target], cwd=module or R, env=env)
    time.sleep(4)


# The Graphics and GUI window helpers and the interactive Plot viewer are their
# own Go modules, so their window and rendering dependencies never enter the
# compiler's module graph. They share AhdCode's own cmd/ahdidentity module (the
# AhdCode name and icon) through a local replace directive.
GRAPHICS = R / 'cmd/ahdgraphics'
GUI = R / 'cmd/ahdgui'
PLOTVIEW = R / 'cmd/ahdplotview'


def first_party(m):
    # A module replaced by a directory inside this repository is AhdCode's own
    # code, covered by LICENSE, not a third-party dependency.
    target = (m.get('Replace') or {}).get('Path', '')
    return target.startswith('../') or target.startswith('./')


def helper_modules(module):
    raw = subprocess.check_output(['go', 'list', '-m', '-json', 'all'], cwd=module, text=True)
    decoder = json.JSONDecoder()
    found = []
    while raw.strip():
        m, end = decoder.raw_decode(raw.lstrip())
        found.append(m)
        raw = raw.lstrip()[end:]
    return found


def licenses(payload, modules):
    dest = payload / 'licenses'
    dest.mkdir()
    for name in [
        'LICENSE',
        'THIRD_PARTY_NOTICES_BCRYPT.md',
        'THIRD_PARTY_NOTICES_CODES.md',
        'THIRD_PARTY_NOTICES_GRAPHICS.md',
        'THIRD_PARTY_NOTICES_GUI.md',
        'THIRD_PARTY_NOTICES_MYSQL.md',
        'THIRD_PARTY_NOTICES_NUMERIC.md',
        'THIRD_PARTY_NOTICES_PLOT.md',
        'THIRD_PARTY_NOTICES_POSTGRESQL.md',
        'THIRD_PARTY_NOTICES_SQLITE.md',
        'THIRD_PARTY_NOTICES_WEBSOCKET.md',
    ]:
        shutil.copy2(R / name, payload / name)
    shutil.copy2(R / 'tooling/distribution/THIRD_PARTY_NOTICES.md', payload / 'THIRD_PARTY_NOTICES.md')
    shutil.copytree(R / 'internal/backend/golang/ahdruntime/mysqlvendor/vendor', dest / 'mysql-source')
    shutil.copytree(R / 'internal/backend/golang/ahdruntime/codesvendor/vendor', dest / 'codes-source')
    (dest / 'bootstrap').mkdir()
    shutil.copy2(R / 'internal/initweb/templates/vendor/bootstrap/LICENSE', dest / 'bootstrap/LICENSE')
    shutil.copy2(R / 'tooling/latex/resources.json', dest / 'latex-resources.json')
    (dest / 'modules' / 'SalvioniDigitalSolutions_gopdf-local').mkdir(parents=True, exist_ok=True)
    shutil.copy2(R / 'third_party/gopdf/LICENSE', dest / 'modules' / 'SalvioniDigitalSolutions_gopdf-local' / 'LICENSE')
    inventory = []
    for mod in modules:
        if mod.get('Main'):
            continue
        path = pathlib.Path(mod.get('Dir', ''))
        if not path.is_dir() or not mod.get('Dir'):
            continue
        entry = {
            'module': mod['Path'],
            'version': mod.get('Version'),
            'source': 'https://pkg.go.dev/' + mod['Path'] + '@' + mod.get('Version', ''),
            'notices': [],
        }
        for file in path.iterdir():
            if file.is_file() and file.name.upper().startswith(('LICENSE', 'COPYING', 'NOTICE', 'PATENTS')):
                rel = pathlib.Path('modules') / (mod['Path'].replace('/', '_') + '@' + mod.get('Version', '')) / file.name
                (dest / rel).parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(file, dest / rel)
                entry['notices'].append(str(rel))
        if not entry['notices']:
            raise RuntimeError('Missing license: ' + mod['Path'])
        inventory.append(entry)
    (dest / 'modules.json').write_text(json.dumps(inventory, indent=2) + '\n')


def zip_payload(payload, out):
    with zipfile.ZipFile(out, 'w', zipfile.ZIP_DEFLATED, compresslevel=6) as z:
        for p in sorted(payload.rglob('*')):
            if p.is_file():
                info = zipfile.ZipInfo(p.relative_to(payload).as_posix(), (2026, 1, 1, 0, 0, 0))
                info.external_attr = (p.stat().st_mode & 0o777) << 16
                info.compress_type = zipfile.ZIP_DEFLATED
                z.writestr(info, p.read_bytes())


# The LaTeX runtime is staged outside the repository, so it is the one payload
# component a release could plausibly acquire in a damaged form: a truncated
# copy, a text-mode transfer, or a Git LFS pointer standing in for the real
# file. Existence is not evidence, so the bundle is checked by magic, size and
# digest, and each engine by its executable format for the platform it ships to.
LFS_POINTER = b'version https://git-lfs.github.com/spec/'
TTB_MAGIC = b'tectonicbundle'
EXECUTABLE_MAGIC = {'darwin': [b'\xcf\xfa\xed\xfe', b'\xca\xfe\xba\xbe'], 'linux': [b'\x7fELF'], 'windows': [b'MZ']}


def check_bundle(path, expected):
    if not path.is_file():
        raise SystemExit('LaTeX bundle is missing: ' + str(path))
    head = path.open('rb').read(len(LFS_POINTER))
    if head.startswith(LFS_POINTER):
        raise SystemExit('LaTeX bundle is a Git LFS pointer, not the bundle: ' + str(path))
    if not head.startswith(TTB_MAGIC):
        raise SystemExit('LaTeX bundle does not start with the Tectonic bundle magic: ' + str(path))
    size = path.stat().st_size
    if size != expected['ttb_size']:
        raise SystemExit('LaTeX bundle is %d bytes, expected %d: %s' % (size, expected['ttb_size'], path))
    digest = sha(path)
    if digest != expected['ttb_sha256']:
        raise SystemExit('LaTeX bundle digest %s does not match the pinned %s' % (digest, expected['ttb_sha256']))


def check_engine(path, goos):
    if not path.is_file():
        raise SystemExit('LaTeX engine is missing: ' + str(path))
    head = path.open('rb').read(4)
    if head.startswith(LFS_POINTER[:4]):
        raise SystemExit('LaTeX engine is a Git LFS pointer: ' + str(path))
    if not any(head.startswith(magic) for magic in EXECUTABLE_MAGIC[goos]):
        raise SystemExit('LaTeX engine is not a %s executable: %s' % (goos, path))
    if path.stat().st_size < 1 << 20:
        raise SystemExit('LaTeX engine is implausibly small: ' + str(path))


def check_vsix(path):
    """The editor extension is built by a separate, network-using step, so the
    platform build treats it as an input to verify rather than trust."""
    if not path.is_file():
        raise SystemExit('VS Code extension package is missing: ' + str(path))
    if not zipfile.is_zipfile(path):
        raise SystemExit('VS Code extension package is not a .vsix archive: ' + str(path))
    with zipfile.ZipFile(path) as archive:
        names = set(archive.namelist())
        for required in ['extension.vsixmanifest', 'extension/package.json', 'extension/extension.js']:
            if required not in names:
                raise SystemExit('VS Code extension package is missing %s: %s' % (required, path))
        manifest = json.loads(archive.read('extension/package.json'))
        if not any(n.startswith('extension/node_modules/vscode-languageclient/') for n in names):
            raise SystemExit('VS Code extension package does not bundle its language client: ' + str(path))
    return manifest['version']


def zip_tree(source, out, root):
    """Archive a staging directory under one top-level folder, with the execute
    bits intact after Finder expands it."""
    renamed = source.parent / root
    source.rename(renamed)
    try:
        run(['ditto', '-c', '-k', '--sequesterRsrc', '--keepParent', renamed, out])
    finally:
        renamed.rename(source)
    verify_zip_modes(out, root)


def verify_zip_modes(archive, root):
    """Expand the way Finder does and insist the entry points are runnable."""
    import tempfile
    with tempfile.TemporaryDirectory(prefix='ahdcode-zipcheck-') as workspace:
        run(['ditto', '-x', '-k', archive, workspace])
        for relative in ['Install.command', 'install.sh', 'payload/bin/ahdcode']:
            path = pathlib.Path(workspace) / root / relative
            if not path.is_file():
                raise SystemExit('the archive is missing ' + relative)
            if not os.access(path, os.X_OK):
                raise SystemExit('Finder would expand ' + relative + ' without its execute bit')


def tar_tree(source, out, arcname):
    """Write a deterministic .tar.gz preserving execute permissions."""
    fixed_mtime = 1767225600  # 2026-01-01T00:00:00Z
    with tarfile.open(out, 'w:gz', format=tarfile.USTAR_FORMAT) as archive:
        for path in [source] + sorted(source.rglob('*')):
            rel = path.relative_to(source)
            member_name = arcname if str(rel) == '.' else f'{arcname}/{rel.as_posix()}'
            info = archive.gettarinfo(path, arcname=member_name)
            info.mtime = fixed_mtime
            info.uid = 0
            info.gid = 0
            info.uname = 'root'
            info.gname = 'wheel'
            if path.is_dir():
                info.mode = 0o755
                archive.addfile(info)
            elif path.is_symlink():
                archive.addfile(info)
            elif path.is_file():
                info.mode = 0o755 if (path.stat().st_mode & 0o111) else 0o644
                with open(path, 'rb') as handle:
                    archive.addfile(info, handle)


MACHO_MAGIC = {b'\xcf\xfa\xed\xfe', b'\xce\xfa\xed\xfe', b'\xca\xfe\xba\xbe'}


def macho_files(root):
    """Every Mach-O file in the payload. Notarization refuses a package that
    carries even one unsigned executable, so this looks at magic numbers rather
    than trusting file names or the execute bit."""
    found = []
    for p in sorted(root.rglob('*')):
        if not p.is_file() or p.is_symlink():
            continue
        with open(p, 'rb') as f:
            head = f.read(4)
        if head in MACHO_MAGIC:
            found.append(p)
    return found


def sign_payload(root, identity):
    """Sign with the hardened runtime and a secure timestamp, which notarization
    requires."""
    targets = macho_files(root)
    if not targets:
        raise SystemExit('no Mach-O files found to sign under ' + str(root))
    for target in targets:
        run(['codesign', '--force', '--options', 'runtime', '--timestamp', '--sign', identity, target])
    for target in targets:
        result = subprocess.run(['codesign', '--verify', '--strict', '--verbose=1', str(target)], capture_output=True, text=True)
        if result.returncode != 0:
            raise SystemExit('signature did not verify: %s\n%s' % (target, result.stderr.strip()))
    print('Signed and verified', len(targets), 'Mach-O files', flush=True)


def dmg_instructions(version, arch, label, commit, pkg_name, bundled_runtimes, sign_app, sign_installer, notary_profile):
    if sign_app and sign_installer and notary_profile:
        trust = 'Status: Signed with Apple Developer ID and notarized by Apple.'
    elif sign_app or sign_installer:
        trust = 'Status: Signed development / release-candidate artifact (NOT notarized by Apple).'
    else:
        trust = (
            'Status: UNSIGNED development / release-candidate artifact.\n'
            'Verify SHA-256 checksums before use. Do NOT disable or bypass Gatekeeper.'
        )

    runtime_note = (
        'Includes private Go toolchain and offline LaTeX engine.'
        if bundled_runtimes
        else 'Compiler + helpers toolchain package (requires Go specified in go.mod on PATH for native builds).'
    )

    arch_note = (
        'Native Apple Silicon (arm64) macOS package.'
        if arch == 'arm64'
        else 'Intel x86-64 (amd64) macOS package (cross-compiled when built on Apple Silicon).'
    )

    return (
        f'AhdCode {version} — macOS Installation Guide ({label})\n'
        f'Commit: {commit[:12]}\n'
        f'{arch_note}\n'
        f'{runtime_note}\n'
        f'{trust}\n\n'
        '1. Recommended: Install using the PKG installer\n'
        f'   Double-click {pkg_name} (or run:\n'
        f'     installer -pkg {pkg_name} -target CurrentUserHomeDirectory\n'
        '   on CI/automated hosts).\n'
        '   - Installs into your home directory only (~/Library/AhdCode), with no administrator password.\n'
        f'   - Places files in ~/Library/AhdCode/versions/{version} and preserves prior versions (including 2.9.0).\n'
        '   - Updates ~/Library/AhdCode/current and ~/Library/AhdCode/bin/ahdcode.\n'
        '   - Idempotently adds ~/Library/AhdCode/bin to ~/.zprofile, ~/.zshrc, and ~/.bash_profile.\n\n'
        '2. Verify checksums\n'
        '   shasum -a 256 -c SHA256SUMS.txt\n\n'
        '3. Roll back to an older installed version (example: 2.9.0)\n'
        '   ln -sfn versions/2.9.0 ~/Library/AhdCode/.current-new && mv -fh ~/Library/AhdCode/.current-new ~/Library/AhdCode/current\n\n'
        '4. Uninstall AhdCode (preserves user projects, databases, and Application Support data)\n'
        '   sh "$HOME/Library/AhdCode/current/install.sh" --uninstall\n'
    )


def source_only_release(version, platform='all'):
    """v2.8.0 through v2.10.x stay source-only; Windows/Linux remain source-only
    for v2.11.x while native macOS RC packaging is enabled on --platform darwin."""
    head = version.split('-')[0]
    parts = head.split('.')
    if len(parts) < 2:
        return False
    try:
        major, minor = int(parts[0]), int(parts[1])
    except ValueError:
        return False
    if major == 2 and minor >= 8:
        if platform == 'darwin' and (major, minor) >= (2, 11):
            return False
        return True
    return major == 3 and minor == 0 and 'beta' in version.lower()


def macos_artifact_suffix(sign_app, sign_installer, notary_profile):
    if sign_app and sign_installer and notary_profile:
        return ''
    if sign_app or sign_installer:
        return '-unnotarized'
    return '-unsigned'


def main():
    version = re.search(r'Number\s*=\s*"([^"]+)"', (R / 'internal/ahdversion/version.go').read_text())[1]
    ap = argparse.ArgumentParser()
    ap.add_argument('--downloads', type=pathlib.Path)
    ap.add_argument('--latex-runtime', type=pathlib.Path)
    ap.add_argument('--modules', type=pathlib.Path)
    ap.add_argument('--output', type=pathlib.Path, required=True)
    ap.add_argument('--sign-app', default='', help='Developer ID Application identity for packaged Mach-O binaries and DMG')
    ap.add_argument('--sign-installer', default='', help='Developer ID Installer identity for .pkg')
    ap.add_argument('--notary-profile', default='', help='xcrun notarytool keychain profile name for signing+notarization')
    ap.add_argument('--vsix', type=pathlib.Path, help='the .vsix from tooling/distribution/build_vsix.py')
    ap.add_argument('--without-vsix', action='store_true', help='deliberately ship no editor extension')
    ap.add_argument('--without-bundled-runtimes', action='store_true', help='macOS only: package compiler and helper binaries without bundled Go/LaTeX downloads')
    ap.add_argument('--platform', choices=['darwin', 'windows', 'linux', 'all'], default='all')
    ap.add_argument('--macos-arch', choices=['arm64', 'amd64', 'both'], default='arm64', help='macOS target architecture(s)')
    ap.add_argument('--leak-forbid-file', type=pathlib.Path, action='append', default=[], help='a developer-local file whose exact contents must not appear in any artifact')
    ap.add_argument('--leak-forbid-env-values', type=pathlib.Path, action='append', default=[], help='a developer .env file whose specific values must not appear in any artifact')
    ap.add_argument('--allow-dirty-tree', action='store_true', help='development only: build from a tree with uncommitted, untracked, or ignored files; never for a release')
    a = ap.parse_args()

    if source_only_release(version, a.platform):
        raise SystemExit(
            'v' + version + ' is a source-only release for ' + a.platform + '.\n'
            'Tag the source and publish a GitHub Release of that source.\n'
            'Do not build a Windows installer or Linux archive, and do not sign or notarize.\n'
            'The v2.7.0 installers stay on their existing GitHub Release and are not rebuilt or retagged.\n'
            'Full multi-platform desktop distribution returns at v3.0.'
        )

    if a.without_bundled_runtimes and a.platform != 'darwin':
        raise SystemExit('--without-bundled-runtimes is supported only for --platform darwin')
    if a.notary_profile and not (a.sign_app and a.sign_installer):
        raise SystemExit('--notary-profile requires both --sign-app and --sign-installer')
    if not a.without_bundled_runtimes and (not a.downloads or not a.latex_runtime):
        raise SystemExit('pass --downloads <dir> and --latex-runtime <dir> (or --without-bundled-runtimes for --platform darwin)')

    commit = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=R, text=True).strip()
    dirty = subprocess.check_output(['git', 'status', '--porcelain', '--ignored', '--untracked-files=all'], cwd=R, text=True)
    if dirty.strip() and not a.allow_dirty_tree:
        raise SystemExit('refusing to build release artifacts from a tree with uncommitted, untracked, or ignored files; build from a clean git worktree:\n' + dirty[:4000])

    if not a.vsix and not a.without_vsix:
        raise SystemExit('pass --vsix <file> (build it with tooling/distribution/build_vsix.py) or --without-vsix')
    vsix_version = check_vsix(a.vsix) if a.vsix else None

    go = json.loads((R / 'tooling/distribution/go-assets.json').read_text())
    latex = json.loads((R / 'tooling/latex/assets.json').read_text())
    bundle = None
    if not a.without_bundled_runtimes:
        bundle = a.latex_runtime / 'ahdcode-latex.ttb'
        check_bundle(bundle, latex['bundle'])

    a.output.mkdir(parents=True, exist_ok=True)
    records = []
    targets = []
    if a.platform in ('all', 'darwin'):
        if a.macos_arch in ('arm64', 'both'):
            targets.append(('darwin', 'arm64', 'macos-arm64'))
        if a.macos_arch in ('amd64', 'both'):
            targets.append(('darwin', 'amd64', 'macos-x64'))
    if a.platform in ('all', 'windows'):
        targets.append(('windows', 'amd64', 'windows-x64'))
    if a.platform in ('all', 'linux'):
        targets.append(('linux', 'amd64', 'linux-x64'))
        targets.append(('linux', 'arm64', 'linux-arm64'))

    for goos, arch, label in targets:
        staging = a.output / ('stage-' + label)
        if staging.exists():
            shutil.rmtree(staging)
        staging.mkdir()
        payload = staging / 'payload'
        (payload / 'bin').mkdir(parents=True)
        (payload / 'libexec/ahdcode').mkdir(parents=True)

        if not a.without_bundled_runtimes:
            asset = next(x for x in go['files'] if x['os'] == goos and x['arch'] == arch)
            archive = a.downloads / asset['filename']
            assert sha(archive) == asset['sha256']
            extract(archive, payload / 'libexec')
            engine = next(x for x in latex['engines'] if x['goos'] == goos and x['goarch'] == arch)
            archive = a.downloads / engine['filename']
            assert sha(archive) == engine['sha256']
            latexdir = payload / 'libexec/ahdcode/latex'
            latexdir.mkdir()
            extract(archive, latexdir)
            shutil.copy2(bundle, latexdir / bundle.name)
            shutil.copy2(R / 'tooling/latex/THIRD_PARTY_NOTICES.txt', latexdir / 'THIRD_PARTY_NOTICES.txt')
            shutil.copytree(R / 'tooling/latex/licenses', latexdir / 'licenses')
            check_bundle(latexdir / bundle.name, latex['bundle'])
            check_engine(latexdir / ('tectonic.exe' if goos == 'windows' else 'tectonic'), goos)

        suffix = '.exe' if goos == 'windows' else ''
        for name in ['ahdcode', 'ahdsqlite', 'ahdnumeric', 'ahdplot']:
            target = payload / ('bin' if name == 'ahdcode' else 'libexec/ahdcode') / (name + suffix)
            build('./cmd/' + name, target, goos, arch)
        build('.', payload / 'libexec/ahdcode' / ('ahdgraphics' + suffix), goos, arch, module=GRAPHICS)
        build('.', payload / 'libexec/ahdcode' / ('ahdgui' + suffix), goos, arch, module=GUI)
        build('.', payload / 'libexec/ahdcode' / ('ahdplotview' + suffix), goos, arch, module=PLOTVIEW)

        if goos == 'windows':
            (payload / 'launcher').mkdir()
            build('./tooling/distribution/windows/launcher', payload / 'launcher/ahdcode.exe', goos, arch)

        if a.modules:
            raw = a.modules.read_text()
            decoder = json.JSONDecoder()
            modules = []
            while raw.strip():
                m, end = decoder.raw_decode(raw.lstrip())
                modules.append(m)
                raw = raw.lstrip()[end:]
        else:
            modules = [m for m in helper_modules(R) if not m.get('Main') and not first_party(m)]

        seen = {(m['Path'], m.get('Version')) for m in modules}
        helpers = []
        for module in [GRAPHICS, GUI, PLOTVIEW]:
            for m in helper_modules(module):
                if (m['Path'], m.get('Version')) not in seen and not m.get('Main') and not first_party(m):
                    seen.add((m['Path'], m.get('Version')))
                    helpers.append(m)
        licenses(payload, modules + helpers)
        shutil.copytree(R / 'internal/initweb/docbundle', payload / 'docs')

        if a.vsix:
            editor = payload / 'vscode'
            editor.mkdir()
            shutil.copy2(a.vsix, editor / a.vsix.name)
            (editor / 'README.txt').write_text(
                'AhdCode for Visual Studio Code\n\n'
                'This folder holds the AhdCode editor extension as a .vsix file. AhdCode\n'
                'itself does not need it, and nothing installs it for you.\n\n'
                'To install it:\n'
                '  1. Open Visual Studio Code.\n'
                '  2. Open the Extensions view.\n'
                '  3. Open the "..." menu at the top of that view.\n'
                '  4. Choose "Install from VSIX..." and select ' + a.vsix.name + '.\n\n'
                'Google Antigravity IDE offers the same "Install from VSIX..." operation.\n\n'
                'The extension runs .ahd files from the editor and connects to the AhdCode\n'
                'language server (ahdcode lsp) for diagnostics and hover. It needs no npm,\n'
                'no network, and no separate download.\n'
            )

        if goos != 'windows':
            shutil.copy2(R / 'tooling/distribution/install.sh', payload / 'install.sh')
            (payload / 'install.sh').chmod(0o755)

        if goos == 'darwin' and a.sign_app:
            sign_payload(payload, a.sign_app)

        (payload / 'VERSION').write_text(version + '\n')
        (staging / 'VERSION').write_text(version + '\n')
        build_meta = {
            'version': version,
            'commit': commit,
            'platform': goos,
            'architecture': arch,
            'bundled_runtimes': not a.without_bundled_runtimes,
            'signed_app': bool(goos == 'darwin' and a.sign_app),
            'signed_installer': bool(goos == 'darwin' and a.sign_installer),
            'notarized': bool(goos == 'darwin' and a.notary_profile),
        }
        if not a.without_bundled_runtimes:
            build_meta['go'] = go['version']
            build_meta['tectonic'] = latex['tectonic_version']
        (payload / 'BUILD.json').write_text(json.dumps(build_meta, indent=2) + '\n')
        if goos == 'windows':
            shutil.copy2(R / 'tooling/distribution/windows/uninstall.ps1', payload / 'uninstall.ps1')
        hashes = {p.relative_to(payload).as_posix(): sha(p) for p in sorted(payload.rglob('*')) if p.is_file()}
        (payload / 'FILES.json').write_text(json.dumps(hashes, indent=2) + '\n')

        base_components = [
            'CLI', 'Studio (embedded)', 'starters (embedded)', 'English docs',
            'ahdsqlite', 'ahdnumeric', 'ahdplot', 'ahdgraphics', 'ahdgui', 'ahdplotview',
        ]
        if not a.without_bundled_runtimes:
            base_components += ['Go ' + go['version'], 'Tectonic ' + latex['tectonic_version'] + ' offline']
        if vsix_version:
            base_components.append('VS Code extension ' + vsix_version)

        if goos == 'windows':
            artifact = a.output / ('AhdCode-' + version + '-' + label + '.exe')
            zipped = R / 'tooling/distribution/windows/payload.zip'
            try:
                zip_payload(payload, zipped)
                build('./tooling/distribution/windows', artifact, goos, arch, '-buildid= -H=windowsgui')
            finally:
                zipped.unlink(missing_ok=True)
            records.append({
                'filename': artifact.name,
                'size': artifact.stat().st_size,
                'sha256': sha(artifact),
                'platform': goos,
                'architecture': arch,
                'format': 'exe',
                'components': base_components + ['graphical per-user setup', 'stable ahdcode.exe launcher'],
                'notices': 'payload/THIRD_PARTY_NOTICES.md',
            })
            print('ARTIFACT', artifact, records[-1]['sha256'], flush=True)
        elif goos == 'darwin':
            art_suffix = macos_artifact_suffix(a.sign_app, a.sign_installer, a.notary_profile)
            pkg = a.output / f'AhdCode-{version}-{label}{art_suffix}.pkg'
            targz = a.output / f'AhdCode-{version}-{label}{art_suffix}.tar.gz'
            dmg = a.output / f'AhdCode-{version}-{label}{art_suffix}.dmg'

            instructions = dmg_instructions(
                version, arch, label, commit, pkg.name, not a.without_bundled_runtimes,
                a.sign_app, a.sign_installer, a.notary_profile,
            )
            shutil.copy2(R / 'tooling/distribution/install.sh', staging / 'install.sh')
            (staging / 'install.sh').chmod(0o755)
            (staging / 'INSTALL.txt').write_text(instructions)
            (staging / 'INSTALL.txt').chmod(0o644)

            tar_tree(staging, targz, f'AhdCode-{version}-{label}')

            pkg_cmd = [
                sys.executable, R / 'tooling/distribution/build_pkg.py',
                '--payload', payload,
                '--version', version,
                '--arch', arch,
                '--output', pkg,
            ]
            if a.sign_installer:
                pkg_cmd += ['--sign', a.sign_installer]
            if a.notary_profile:
                pkg_cmd += ['--notary-profile', a.notary_profile]
            for path in a.leak_forbid_file:
                pkg_cmd += ['--leak-forbid-file', path]
            for path in a.leak_forbid_env_values:
                pkg_cmd += ['--leak-forbid-env-values', path]
            run(pkg_cmd)

            dmg_stage = a.output / f'dmg-{label}'
            if dmg_stage.exists():
                shutil.rmtree(dmg_stage)
            dmg_stage.mkdir()
            shutil.copy2(pkg, dmg_stage / pkg.name)
            (dmg_stage / 'VERSION').write_text(version + '\n')
            (dmg_stage / 'INSTALL.txt').write_text(instructions)
            volname = f'AhdCode {version} {label}' + ('' if art_suffix == '' else f' {art_suffix.lstrip("-")}')
            run(['hdiutil', 'create', '-volname', volname, '-srcfolder', dmg_stage, '-format', 'UDZO', '-ov', dmg])
            if a.sign_app:
                run(['codesign', '--force', '--timestamp', '--sign', a.sign_app, dmg])
                run(['codesign', '--verify', '--verbose=1', dmg])
            if a.sign_app and a.notary_profile:
                run(['xcrun', 'notarytool', 'submit', str(dmg), '--keychain-profile', a.notary_profile, '--wait'])
                run(['xcrun', 'stapler', 'staple', str(dmg)])
                run(['xcrun', 'stapler', 'validate', str(dmg)])

            for fmt, art_path in [('tar.gz', targz), ('pkg', pkg), ('dmg', dmg)]:
                rec = {
                    'filename': art_path.name,
                    'size': art_path.stat().st_size,
                    'sha256': sha(art_path),
                    'platform': goos,
                    'architecture': arch,
                    'format': fmt,
                    'signed': bool(a.sign_installer if fmt == 'pkg' else a.sign_app),
                    'notarized': bool(a.notary_profile and fmt in ('pkg', 'dmg')),
                    'bundled_runtimes': not a.without_bundled_runtimes,
                    'components': base_components + (['home-directory .pkg installer'] if fmt in ('pkg', 'dmg') else ['install.sh installer']),
                    'notices': 'payload/THIRD_PARTY_NOTICES.md',
                }
                records.append(rec)
                print('ARTIFACT', art_path, rec['sha256'], flush=True)
        else:
            artifact = a.output / ('AhdCode-' + version + '-' + label + '.tar.gz')
            shutil.copy2(R / 'tooling/distribution/install.sh', staging / 'install.sh')
            with tarfile.open(artifact, 'w:gz') as t:
                t.add(staging, arcname='AhdCode-' + version)
            records.append({
                'filename': artifact.name,
                'size': artifact.stat().st_size,
                'sha256': sha(artifact),
                'platform': goos,
                'architecture': arch,
                'format': 'tar.gz',
                'components': base_components,
                'notices': 'payload/THIRD_PARTY_NOTICES.md',
            })
            print('ARTIFACT', artifact, records[-1]['sha256'], flush=True)

        (a.output / ('manifest-' + goos + '.json')).write_text(
            json.dumps({'version': version, 'commit': commit, 'artifacts': [r for r in records if r['platform'] == goos]}, indent=2) + '\n'
        )

    if a.vsix and a.vsix.resolve() != (a.output / a.vsix.name).resolve():
        shutil.copy2(a.vsix, a.output / a.vsix.name)

    sums_lines = [f"{r['sha256']}  {r['filename']}\n" for r in sorted(records, key=lambda x: x['filename'])]
    (a.output / 'SHA256SUMS.txt').write_text(''.join(sums_lines))
    (a.output / 'release-manifest.json').write_text(
        json.dumps({'version': version, 'commit': commit, 'extension': vsix_version, 'artifacts': records}, indent=2) + '\n'
    )

    studio = a.output / 'studio-files.txt'
    studio.write_text(subprocess.check_output(['git', 'ls-files', 'tools/AhdDataStudio'], cwd=R, text=True))
    gate = [sys.executable, R / 'tooling/distribution/leak_gate.py', '--studio-files', studio, '--manifest-dir', a.output / 'manifests']
    for path in a.leak_forbid_file:
        gate += ['--forbid-file', path]
    for path in a.leak_forbid_env_values:
        gate += ['--forbid-env-values', path]
    run(gate + [a.output / r['filename'] for r in records] + ([a.output / a.vsix.name] if a.vsix else []))


if __name__ == '__main__':
    main()
