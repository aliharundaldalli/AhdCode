#!/usr/bin/env python3
"""Package the VS Code extension into an installable .vsix release artifact.

This is deliberately separate from build.py. Producing a .vsix needs Node and,
the first time, the npm registry; the platform build must stay offline and
reproducible from pre-verified archives. So the extension is packaged once, and
build.py consumes the finished file the same way it consumes the Go and
Tectonic archives.

Nothing here publishes to the Marketplace and nothing installs into an editor.
"""
import argparse
import json
import pathlib
import shutil
import subprocess
import tempfile
import zipfile

ROOT = pathlib.Path(__file__).resolve().parents[2]
EXTENSION = ROOT / 'editors/vscode'


def run(arguments, cwd):
    subprocess.run([str(a) for a in arguments], cwd=str(cwd), check=True)


def verify(vsix, version):
    """A .vsix is a zip with a fixed layout. Check the parts an editor needs
    before the file is handed to a user, so a truncated or mis-built archive
    cannot reach a release."""
    if not vsix.is_file():
        raise SystemExit('no .vsix was produced at ' + str(vsix))
    if not zipfile.is_zipfile(vsix):
        raise SystemExit('the .vsix is not a zip archive: ' + str(vsix))
    with zipfile.ZipFile(vsix) as archive:
        if archive.testzip() is not None:
            raise SystemExit('the .vsix contains a corrupt entry: ' + str(vsix))
        names = set(archive.namelist())
        for required in ['extension.vsixmanifest', '[Content_Types].xml',
                         'extension/package.json', 'extension/extension.js',
                         'extension/language-configuration.json',
                         'extension/syntaxes/ahdcode.tmLanguage.json']:
            if required not in names:
                raise SystemExit('the .vsix is missing ' + required)
        manifest = json.loads(archive.read('extension/package.json'))
        if manifest['version'] != version:
            raise SystemExit('the .vsix declares version %s, expected %s' % (manifest['version'], version))
        if manifest['name'] != 'ahdcode':
            raise SystemExit('the .vsix declares an unexpected extension name: ' + manifest['name'])
        # The language client is a runtime dependency, so it has to travel
        # inside the package: an end user must never need npm.
        if not any(name.startswith('extension/node_modules/vscode-languageclient/') for name in names):
            raise SystemExit('the .vsix does not bundle vscode-languageclient')
        declaration = archive.read('extension.vsixmanifest').decode('utf-8', 'replace')
        if 'Version="%s"' % version not in declaration:
            raise SystemExit('extension.vsixmanifest does not declare version ' + version)
    return len(names)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', type=pathlib.Path, required=True,
                        help='directory the .vsix is written to')
    parser.add_argument('--offline', action='store_true',
                        help='fail rather than reach the npm registry')
    arguments = parser.parse_args()

    version = json.loads((EXTENSION / 'package.json').read_text())['version']
    arguments.output.mkdir(parents=True, exist_ok=True)
    target = arguments.output / ('ahdcode-' + version + '.vsix')

    # Package from a copy so the repository keeps no node_modules or build
    # leftovers, and so .vscodeignore decides the contents rather than whatever
    # happens to be lying in the working tree.
    with tempfile.TemporaryDirectory(prefix='ahdcode-vsix-') as staging:
        workspace = pathlib.Path(staging) / 'vscode'
        shutil.copytree(EXTENSION, workspace, ignore=shutil.ignore_patterns('node_modules', '*.vsix', '.vscode'))
        install = ['npm', 'ci', '--no-audit', '--no-fund']
        if arguments.offline:
            install.append('--offline')
        run(install, workspace)
        run(['npx', '--yes', '@vscode/vsce', 'package', '--out', target.resolve()], workspace)

    entries = verify(target, version)
    print('Packaged %s (%d entries, %d bytes)' % (target.name, entries, target.stat().st_size))
    print(target)


if __name__ == '__main__':
    main()
