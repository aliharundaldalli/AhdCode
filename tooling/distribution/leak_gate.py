#!/usr/bin/env python3
"""Release leak gate: refuse artifacts that carry developer-local files or data.

The first v1.4.0 release candidate was built from a developer checkout, and the
CLI's `go:embed all:AhdDataStudio` carried an untracked tools/AhdDataStudio/.env
into every artifact. This gate inspects what an artifact actually delivers:
every member of a ZIP, VSIX, or tar archive; the mounted disk image; the fully
expanded installer package; the Windows setup program and the payload it
carries; archives nested inside any of those (such as the VSIX); and the bytes
of every file, including the CLI binary with the trees it embeds.

It fails the artifact when it finds:

- a file named .env or .env.<anything> (only .env.example is allowed);
- an embedded file-system entry AhdDataStudio/.env or AhdDataStudio/.DS_Store;
- developer paths: /Users/ahd, /private/tmp/, /var/folders/, claude-501,
  /scratchpad/, or .claude/worktrees;
- a home-directory path such as /Users/<name>/Desktop/ or
  /home/<name>/projects/ outside the unmodified third-party Go distribution
  and module license texts (inside those it is reported for review);
- exact developer-local data supplied at release time: the whole contents of a
  --forbid-file, and the specific values assigned in a --forbid-env-values
  file. Values are never printed. Generic values such as localhost or root,
  which occur in legitimate documentation, are skipped by key name.

With --studio-files (the output of `git ls-files tools/AhdDataStudio`), every
CLI binary must also embed each tracked AhdDataStudio file.

usage:
  leak_gate.py [--forbid-file PATH]... [--forbid-env-values PATH]...
               [--studio-files PATH] [--manifest-dir DIR] ARTIFACT...
  leak_gate.py --self-test
"""
import argparse
import hashlib
import io
import pathlib
import re
import struct
import subprocess
import sys
import tarfile
import tempfile
import zipfile
import zlib

ALLOWED_ENV_NAMES = {'.env.example'}
GENERIC_VALUES = {'localhost', '127.0.0.1', '::1', '0.0.0.0', 'root', 'admin', 'user', 'none', 'tls',
                  'true', 'false', 'on', 'off', 'yes', 'no'}
STRICT_PATHS = re.compile(rb'/Users/ahd|/private/tmp/|/var/folders/|claude-501|/scratchpad/|\.claude/worktrees')
HOME_PATHS = re.compile(
    rb'/Users/[A-Za-z0-9._-]+/(?:Developer|Desktop|Documents|Downloads|Library|Projects)/'
    rb'|/home/[a-z_][a-z0-9_-]*/(?:Developer|Desktop|Documents|Downloads|projects|src)/')
# Exact documentation examples that match HOME_PATHS on purpose. Each entry is a
# whole example string, never a pattern, so a real developer path still fails.
ALLOWED_HOME_PATH_EXAMPLES = (
    b'/home/ada/projects/ahd/app.ahd',  # docs/CLI.md: an illustrative project path for a fictional user
)
EMBEDDED_STRAY = re.compile(rb'AhdDataStudio/(?:[\w.-]+/)*(?:\.env(?!\.example)|\.DS_Store)')
NESTED_ARCHIVES = ('.zip', '.vsix')
THIRD_PARTY_MARKERS = ('libexec/go/', 'licenses/modules/')
CLI_NAMES = ('bin/ahdcode', 'bin/ahdcode.exe')


class GateError(Exception):
    pass


def load_forbidden(files, env_files):
    """Return [(label, bytes)] and the key names skipped as generic."""
    forbidden, skipped = [], []
    for name in files:
        path = pathlib.Path(name)
        raw = path.read_bytes()
        if raw:
            forbidden.append(('the whole contents of ' + path.name, raw))
        if raw.strip() and raw.strip() != raw:
            forbidden.append(('the trimmed contents of ' + path.name, raw.strip()))
    for name in env_files:
        path = pathlib.Path(name)
        for line in path.read_text(errors='replace').splitlines():
            stripped = line.strip()
            if not stripped or stripped.startswith('#') or '=' not in stripped:
                continue
            key, value = stripped.split('=', 1)
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            if not value:
                continue
            if value.lower() in GENERIC_VALUES or value.isdigit() or len(value) < 6:
                skipped.append(key)
                continue
            forbidden.append(('the value of %s from %s' % (key, path.name), value.encode()))
    return forbidden, skipped


def embedded_zip(data):
    """The payload a Windows setup program carries after its executable code."""
    position = len(data)
    while True:
        position = data.rfind(b'PK\x05\x06', 0, position)
        if position < 0:
            return None
        cd_size, cd_offset, comment = struct.unpack('<IIH', data[position + 12:position + 22])
        start = position - cd_size - cd_offset
        if start < 0 or data[start:start + 4] != b'PK\x03\x04':
            continue
        try:
            archive = zipfile.ZipFile(io.BytesIO(data[start:position + 22 + comment]))
        except zipfile.BadZipFile:
            continue
        if len(archive.namelist()) > 100:
            return archive


def zip_members(archive, prefix):
    for info in archive.infolist():
        if not info.is_dir():
            yield prefix + info.filename, archive.read(info)


def expand(name, data):
    """Yield a member, then every member of the archive it is, recursively.

    A nested archive that cannot be opened -- the Go distribution ships
    deliberately malformed ZIP files as test data -- yields (name + '!', None)
    so the caller decides whether that is acceptable for its location."""
    yield name, data
    if name.lower().endswith(NESTED_ARCHIVES) and zipfile.is_zipfile(io.BytesIO(data)):
        try:
            with zipfile.ZipFile(io.BytesIO(data)) as archive:
                members = list(zip_members(archive, name + '!'))
        except (zipfile.BadZipFile, zlib.error, NotImplementedError, EOFError, ValueError, RuntimeError, OSError):
            yield name + '!', None
            return
        for inner, content in members:
            yield from expand(inner, content)


def tree_members(root):
    for path in sorted(root.rglob('*')):
        relative = path.relative_to(root).as_posix()
        if path.is_symlink():
            yield relative, b''
        elif path.is_file():
            try:
                content = path.read_bytes()
            except OSError as error:
                raise GateError('cannot read %s: %s' % (relative, error))
            yield from expand(relative, content)


def artifact_members(path, workspace):
    lower = path.name.lower()
    if lower.endswith(('.zip', '.vsix')):
        with zipfile.ZipFile(path) as archive:
            for name, data in zip_members(archive, ''):
                yield from expand(name, data)
    elif lower.endswith(('.tar.gz', '.tgz', '.tar')):
        with tarfile.open(path) as archive:
            for member in archive.getmembers():
                if member.isfile():
                    yield from expand(member.name, archive.extractfile(member).read())
                elif member.issym() or member.islnk():
                    yield member.name, b''
    elif lower.endswith('.exe'):
        data = path.read_bytes()
        yield path.name, data
        archive = embedded_zip(data)
        if archive is None:
            raise GateError('no embedded payload found in ' + path.name)
        for name, content in zip_members(archive, path.name + '!'):
            yield from expand(name, content)
    elif lower.endswith('.dmg'):
        mount = tempfile.mkdtemp(prefix='leakgate-dmg-', dir=workspace)
        subprocess.run(['hdiutil', 'attach', '-readonly', '-nobrowse', '-noautoopen', '-mountpoint', mount, str(path)],
                       check=True, capture_output=True)
        try:
            yield from tree_members(pathlib.Path(mount))
        finally:
            subprocess.run(['hdiutil', 'detach', mount], check=False, capture_output=True)
    elif lower.endswith('.pkg'):
        expanded = pathlib.Path(tempfile.mkdtemp(prefix='leakgate-pkg-', dir=workspace)) / 'expanded'
        subprocess.run(['pkgutil', '--expand-full', str(path), str(expanded)], check=True, capture_output=True)
        yield from tree_members(expanded)
    else:
        raise GateError('unsupported artifact type: ' + path.name)


def scan(path, forbidden, studio_files=None, manifest=None):
    """Return (failures, reviews, files, bytes, clis) for one artifact."""
    failures, reviews = [], []
    files = size = clis = 0
    with tempfile.TemporaryDirectory(prefix='leakgate-') as workspace:
        for name, data in artifact_members(path, workspace):
            if data is None:
                target = reviews if any(marker in name for marker in THIRD_PARTY_MARKERS) else failures
                target.append('a nested archive that cannot be opened: ' + name.rstrip('!'))
                continue
            files += 1
            size += len(data)
            if manifest is not None:
                manifest.write('%s  %d  %s\n' % (hashlib.sha256(data).hexdigest(), len(data), name))
            base = name.rsplit('!', 1)[-1].rsplit('/', 1)[-1]
            if (base == '.env' or base.startswith('.env.')) and base not in ALLOWED_ENV_NAMES:
                failures.append('a file named %s: %s' % (base, name))
            if not data:
                continue
            if EMBEDDED_STRAY.search(data):
                failures.append('an embedded AhdDataStudio .env or .DS_Store entry inside ' + name)
            strict = len(STRICT_PATHS.findall(data))
            if strict:
                failures.append('%d developer path occurrence(s) inside %s' % (strict, name))
            home = sum(1 for match in HOME_PATHS.finditer(data)
                       if not any(data.startswith(example, match.start()) for example in ALLOWED_HOME_PATH_EXAMPLES))
            if home:
                target = reviews if any(marker in name for marker in THIRD_PARTY_MARKERS) else failures
                target.append('%d home-directory path occurrence(s) inside %s' % (home, name))
            for label, value in forbidden:
                if value in data:
                    failures.append('%s inside %s' % (label, name))
            if studio_files is not None and name.replace('\\', '/').endswith(CLI_NAMES):
                clis += 1
                missing = [f for f in studio_files if ('AhdDataStudio/' + f).encode() not in data]
                if missing:
                    failures.append('%s does not embed %d tracked AhdDataStudio file(s): %s' % (name, len(missing), missing[:5]))
    return failures, reviews, files, size, clis


def studio_list(path):
    prefix = 'tools/AhdDataStudio/'
    entries = [line.strip() for line in pathlib.Path(path).read_text().splitlines() if line.strip()]
    return [entry[len(prefix):] for entry in entries if entry.startswith(prefix)]


def run(arguments):
    forbidden, skipped = load_forbidden(arguments.forbid_file, arguments.forbid_env_values)
    studio = studio_list(arguments.studio_files) if arguments.studio_files else None
    if skipped:
        print('generic values skipped (they occur in legitimate documentation): ' + ', '.join(skipped))
    print('exact developer-local byte strings checked: %d' % len(forbidden))
    if arguments.manifest_dir:
        arguments.manifest_dir.mkdir(parents=True, exist_ok=True)
    failed = False
    for artifact in arguments.artifacts:
        manifest = None
        if arguments.manifest_dir:
            manifest = (arguments.manifest_dir / (artifact.name + '.files.txt')).open('w')
        try:
            failures, reviews, files, size, clis = scan(artifact, forbidden, studio, manifest)
        except (GateError, subprocess.CalledProcessError, OSError, zipfile.BadZipFile, tarfile.TarError) as error:
            failures, reviews, files, size, clis = ['could not inspect the artifact: %s' % error], [], 0, 0, 0
        finally:
            if manifest is not None:
                manifest.close()
        status = 'FAIL' if failures else 'PASS'
        failed = failed or bool(failures)
        detail = '%d files, %.1f MiB inspected' % (files, size / 1048576)
        if studio is not None:
            detail += ', %d CLI binaries checked against %d tracked AhdDataStudio files' % (clis, len(studio))
        print('%s %s: %s' % (status, artifact.name, detail))
        for failure in failures[:40]:
            print('  FAIL ' + failure)
        if len(failures) > 40:
            print('  ... %d more failures' % (len(failures) - 40))
        for review in reviews[:20]:
            print('  REVIEW (third-party) ' + review)
    return 1 if failed else 0


def self_test():
    with tempfile.TemporaryDirectory(prefix='leakgate-selftest-') as directory:
        root = pathlib.Path(directory)
        secret = root / 'developer.env'
        secret.write_text('AHD_DATA_MYSQL_HOST=localhost\nAHD_DATA_SQLITE_PATHS=/opt/private/area51.db\n')
        forbidden, skipped = load_forbidden([secret], [secret])
        assert skipped == ['AHD_DATA_MYSQL_HOST'], skipped

        def make_zip(name, members):
            path = root / name
            with zipfile.ZipFile(path, 'w') as archive:
                for member, content in members.items():
                    archive.writestr(member, content)
            return path

        inner = io.BytesIO()
        with zipfile.ZipFile(inner, 'w') as archive:
            archive.writestr('extension/.env', 'X=1\n')
        cases = {
            'clean.zip': (make_zip('clean.zip', {
                'payload/.env.example': 'EXAMPLE=1\n',
                'payload/docs/README.md': 'connect to localhost as root; pgx uses /private/tmp as a socket directory',
                'payload/docs/CLI.md': 'source: /home/ada/projects/ahd/app.ahd',
                'payload/bin/ahdcode': b'...AhdDataStudio/app.ahdAhdDataStudio/.env.exampleAhdDataStudio/Shared/Config.ahd...',
            }), False),
            'env-name.zip': (make_zip('env-name.zip', {'payload/tools/AhdDataStudio/.env': 'X=1\n'}), True),
            'embedded.zip': (make_zip('embedded.zip', {'payload/bin/ahdcode': b'AhdDataStudio/app.ahdAhdDataStudio/.envAhdDataStudio/.env.example'}), True),
            'developer-path.zip': (make_zip('developer-path.zip', {'payload/bin/ahdcode': b'watching /Users/ahd/Developer/AhdCode'}), True),
            'home-path.zip': (make_zip('home-path.zip', {'payload/docs/NOTE.md': b'see /Users/someone/Desktop/notes'}), True),
            'home-path-near-example.zip': (make_zip('home-path-near-example.zip', {'payload/docs/NOTE.md': b'source: /home/ada/projects/secret/app.ahd'}), True),
            'value.zip': (make_zip('value.zip', {'payload/bin/ahdcode': b'x /opt/private/area51.db y'}), True),
            'nested.zip': (make_zip('nested.zip', {'payload/vscode/ahdcode.vsix': inner.getvalue()}), True),
        }
        corrupt = io.BytesIO()
        with zipfile.ZipFile(corrupt, 'w') as archive:
            archive.writestr('member.txt', 'content')
        corrupt = corrupt.getvalue().replace(b'PK\x03\x04', b'XX\x03\x04', 1)
        assert zipfile.is_zipfile(io.BytesIO(corrupt))
        cases['malformed-third-party.zip'] = (make_zip('malformed-third-party.zip', {
            'payload/libexec/go/src/archive/zip/testdata/bad.zip': corrupt,
            'payload/docs/README.md': 'clean',
        }), False)
        cases['malformed-owned.zip'] = (make_zip('malformed-owned.zip', {'payload/vscode/ahdcode.vsix': corrupt}), True)
        tar_path = root / 'value.tar.gz'
        with tarfile.open(tar_path, 'w:gz') as archive:
            data = secret.read_bytes()
            info = tarfile.TarInfo('AhdCode/payload/bin/ahdcode')
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
        cases['value.tar.gz'] = (tar_path, True)
        for label, (path, should_fail) in cases.items():
            failures, _, _, _, _ = scan(path, forbidden, ['app.ahd', 'Shared/Config.ahd'] if label == 'clean.zip' else None)
            assert bool(failures) == should_fail, (label, failures)
            for failure in failures:
                assert '/opt/private/area51.db' not in failure, 'a forbidden value was printed: ' + label
    print('leak gate self-test passed')
    return 0


def main():
    parser = argparse.ArgumentParser(description='Refuse release artifacts that carry developer-local files or data.')
    parser.add_argument('artifacts', nargs='*', type=pathlib.Path)
    parser.add_argument('--forbid-file', action='append', default=[], type=pathlib.Path,
                        help='a developer-local file whose exact contents must not appear in any artifact')
    parser.add_argument('--forbid-env-values', action='append', default=[], type=pathlib.Path,
                        help='a .env file whose specific assigned values must not appear in any artifact')
    parser.add_argument('--studio-files', type=pathlib.Path,
                        help='output of `git ls-files tools/AhdDataStudio`; every CLI binary must embed each file')
    parser.add_argument('--manifest-dir', type=pathlib.Path, help='write one inspected-file manifest per artifact here')
    parser.add_argument('--self-test', action='store_true')
    arguments = parser.parse_args()
    if arguments.self_test:
        return self_test()
    if not arguments.artifacts:
        parser.error('pass at least one artifact')
    return run(arguments)


if __name__ == '__main__':
    sys.exit(main())
