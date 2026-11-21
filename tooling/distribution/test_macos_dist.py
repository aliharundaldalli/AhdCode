#!/usr/bin/env python3
"""Regression tests for AhdCode macOS distribution artifacts (.tar.gz, .pkg, .dmg)."""
import hashlib
import json
import os
import pathlib
import plistlib
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
from xml.etree import ElementTree

ROOT = pathlib.Path(__file__).resolve().parents[2]
TOOLS = ROOT / 'tooling' / 'distribution'
EXPECTED_HELPERS = ('ahdsqlite', 'ahdnumeric', 'ahdplot', 'ahdgraphics', 'ahdgui', 'ahdplotview')
DIST_DIR_ENV = os.environ.get('AHDCODE_TEST_MACOS_DIST_DIR', '').strip()


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, 'rb') as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def seed_isolated_home(home):
    """Create an isolated fake $HOME containing a pre-existing 2.9.0 installation,
    pre-existing shell config, and unrelated user data that must survive install/uninstall."""
    lib = home / 'Library'
    ahd_root = lib / 'AhdCode'
    v29_bin = ahd_root / 'versions' / '2.9.0' / 'bin'
    v29_bin.mkdir(parents=True)
    stub = v29_bin / 'ahdcode'
    stub.write_text('#!/bin/sh\necho "AhdCode v2.9.0"\n')
    stub.chmod(0o755)
    (ahd_root / 'versions' / '2.9.0' / 'sentinel.txt').write_text('keep-2.9.0\n')
    (ahd_root / '.ahdcode-install').write_text('AhdCode installer v1\n')
    (ahd_root / 'current').symlink_to('versions/2.9.0')
    (ahd_root / 'bin').mkdir()
    (ahd_root / 'bin' / 'ahdcode').symlink_to('../current/bin/ahdcode')

    support = lib / 'Application Support' / 'ahdcode'
    support.mkdir(parents=True)
    (support / 'user.db').write_text('sentinel-data\n')

    (home / '.zshrc').write_text('# user-config-keep\n')
    return ahd_root, support / 'user.db'


def verify_sqlite_execution(ahdcode_bin, workdir):
    """Run a real SQLite program through installed ahdcode without AHDCODE_SQLITE_RUNTIME."""
    env = dict(os.environ)
    env.pop('AHDCODE_SQLITE_RUNTIME', None)
    env.pop('AHDCODE_ROOT', None)
    proj = workdir / 'sqlite_smoke'
    proj.mkdir()
    src = proj / 'main.ahd'
    src.write_text(
        'bring SQLite\n'
        'from SQLite bring (Database, SQLiteValue)\n'
        'db: Database := SQLite.open("test.db")\n'
        'db.execute("CREATE TABLE items(id INTEGER PRIMARY KEY, name TEXT)")\n'
        'db.execute("INSERT INTO items(name) VALUES(?)", [SQLite.fromString("rc1-ok")])\n'
        'rows: List<Pair<String, SQLiteValue>> := db.query("SELECT name FROM items")\n'
        'write(rows[0]["name"].string())\n'
        'db.close()\n'
    )
    res = subprocess.run([str(ahdcode_bin), 'run', str(src)], cwd=proj, env=env, capture_output=True, text=True, check=True)
    if res.stdout.strip() != 'rc1-ok':
        raise AssertionError(f'unexpected SQLite run output: {res.stdout!r} stderr={res.stderr!r}')


class FastDistributionPolicyTests(unittest.TestCase):
    def test_source_only_gate_preserves_windows_linux_behavior(self):
        for plat in ('windows', 'linux', 'all'):
            with tempfile.TemporaryDirectory(prefix='ahdcode-pol-') as tmp:
                res = subprocess.run(
                    [sys.executable, str(TOOLS / 'build.py'), '--platform', plat, '--without-vsix', '--output', tmp],
                    cwd=ROOT, capture_output=True, text=True,
                )
                self.assertNotEqual(res.returncode, 0, f'platform={plat} should still be blocked on 2.11.0')
                self.assertIn('source-only release', res.stderr)

    def test_postinstall_rejects_traversal_and_symlink_root(self):
        postinstall = TOOLS / 'macos' / 'postinstall'
        with tempfile.TemporaryDirectory(prefix='ahdcode-post-') as tmp:
            base = pathlib.Path(tmp)
            for bad_ver in ('', '../escape', '2.11/bad', '2.11;rm'):
                fake_home = base / ('home_' + str(abs(hash(bad_ver))))
                vdir = fake_home / 'Library' / 'AhdCode' / 'versions'
                vdir.mkdir(parents=True)
                (vdir / 'VERSION.current').write_text(bad_ver)
                res = subprocess.run(
                    ['sh', str(postinstall), '', '', str(fake_home)],
                    env={'HOME': str(fake_home), 'PATH': os.environ['PATH']},
                    capture_output=True, text=True,
                )
                self.assertNotEqual(res.returncode, 0, f'postinstall accepted bad version {bad_ver!r}')
                self.assertFalse((fake_home / 'Library' / 'AhdCode' / 'current').exists())

            sym_home = base / 'sym_home'
            outside = base / 'outside'
            outside.mkdir()
            (sym_home / 'Library').mkdir(parents=True)
            (sym_home / 'Library' / 'AhdCode').symlink_to(outside)
            res = subprocess.run(
                ['sh', str(postinstall), '', '', str(sym_home)],
                env={'HOME': str(sym_home), 'PATH': os.environ['PATH']},
                capture_output=True, text=True,
            )
            self.assertNotEqual(res.returncode, 0)

    def test_leak_gate_self_test(self):
        subprocess.run([sys.executable, str(TOOLS / 'leak_gate.py'), '--self-test'], check=True, capture_output=True, text=True)


@unittest.skipUnless(sys.platform == 'darwin' and bool(DIST_DIR_ENV), 'Set AHDCODE_TEST_MACOS_DIST_DIR on macOS to exercise generated artifacts')
class MacOSDistributionArtifactTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.dist_dir = pathlib.Path(DIST_DIR_ENV)
        cls.manifest = json.loads((cls.dist_dir / 'manifest-darwin.json').read_text())
        cls.version = cls.manifest['version']

    def _artifact(self, arch, fmt):
        for rec in self.manifest['artifacts']:
            if rec['architecture'] == arch and rec['format'] == fmt:
                return self.dist_dir / rec['filename'], rec
        self.fail(f'missing artifact architecture={arch} format={fmt}')

    def test_sha256sums_and_manifests(self):
        sums_path = self.dist_dir / 'SHA256SUMS.txt'
        self.assertTrue(sums_path.is_file())
        parsed = {}
        for line in sums_path.read_text().splitlines():
            if not line.strip():
                continue
            digest, filename = line.split('  ', 1)
            parsed[filename] = digest
            self.assertEqual(sha256_file(self.dist_dir / filename), digest)
        for fmt in ('tar.gz', 'pkg', 'dmg'):
            path, rec = self._artifact('arm64', fmt)
            self.assertTrue(path.is_file())
            self.assertEqual(parsed[path.name], rec['sha256'])
            self.assertEqual(sha256_file(path), rec['sha256'])

    def test_targz_install_execute_upgrade_rollback_uninstall(self):
        targz, _ = self._artifact('arm64', 'tar.gz')
        with tempfile.TemporaryDirectory(prefix='ahdcode-tgz-test-') as tmp:
            work = pathlib.Path(tmp)
            fake_home = work / 'home'
            ahd_root, user_db = seed_isolated_home(fake_home)
            extracted = work / 'extracted'
            extracted.mkdir()
            with tarfile.open(targz, 'r:gz') as tf:
                tf.extractall(extracted, filter='data')
            root_dirs = list(extracted.iterdir())
            self.assertEqual(len(root_dirs), 1)
            installer_dir = root_dirs[0]
            self.assertTrue((installer_dir / 'install.sh').is_file())
            self.assertTrue((installer_dir / 'INSTALL.txt').is_file())
            self.assertEqual((installer_dir / 'VERSION').read_text().strip(), self.version)

            env = {'HOME': str(fake_home), 'PATH': os.environ['PATH']}
            subprocess.run(['sh', str(installer_dir / 'install.sh'), '--setup-path'], env=env, check=True, capture_output=True, text=True)

            # Verify 2.9.0 preserved, current points to new version, binaries executable.
            self.assertEqual((ahd_root / 'versions' / '2.9.0' / 'sentinel.txt').read_text().strip(), 'keep-2.9.0')
            self.assertEqual(os.readlink(ahd_root / 'current'), f'versions/{self.version}')
            self.assertEqual(os.readlink(ahd_root / 'bin' / 'ahdcode'), '../current/bin/ahdcode')
            ver_dir = ahd_root / 'versions' / self.version
            self.assertTrue(os.access(ver_dir / 'bin' / 'ahdcode', os.X_OK))
            self.assertTrue(os.access(ver_dir / 'install.sh', os.X_OK))
            for helper in EXPECTED_HELPERS:
                self.assertTrue(os.access(ver_dir / 'libexec' / 'ahdcode' / helper, os.X_OK), f'missing/non-exec {helper}')

            # Verify PATH block added once and idempotent on re-run.
            subprocess.run(['sh', str(installer_dir / 'install.sh'), '--setup-path'], env=env, check=True, capture_output=True, text=True)
            for profile_name in ('.zprofile', '.zshrc', '.bash_profile'):
                txt = (fake_home / profile_name).read_text()
                self.assertEqual(txt.count('# >>> AhdCode PATH >>>'), 1, profile_name)
            self.assertIn('# user-config-keep', (fake_home / '.zshrc').read_text())

            # Real execution checks.
            ver_out = subprocess.run([str(ahd_root / 'bin' / 'ahdcode'), 'version'], check=True, capture_output=True, text=True).stdout.strip()
            self.assertEqual(ver_out, f'AhdCode v{self.version}')
            verify_sqlite_execution(ahd_root / 'bin' / 'ahdcode', work)

            # Rollback to 2.9.0 and switch back to 2.11.0.
            (ahd_root / '.current-new').symlink_to('versions/2.9.0')
            os.replace(ahd_root / '.current-new', ahd_root / 'current')
            self.assertEqual(subprocess.run([str(ahd_root / 'bin' / 'ahdcode'), '--version'], check=True, capture_output=True, text=True).stdout.strip(), 'AhdCode v2.9.0')
            (ahd_root / '.current-new').symlink_to(f'versions/{self.version}')
            os.replace(ahd_root / '.current-new', ahd_root / 'current')
            self.assertEqual(subprocess.run([str(ahd_root / 'bin' / 'ahdcode'), 'version'], check=True, capture_output=True, text=True).stdout.strip(), f'AhdCode v{self.version}')

            # Isolated custom --prefix check (does not touch shell profiles).
            custom_prefix = work / 'custom-prefix'
            subprocess.run(['sh', str(installer_dir / 'install.sh'), '--prefix', str(custom_prefix)], env=env, check=True, capture_output=True, text=True)
            self.assertEqual(subprocess.run([str(custom_prefix / 'bin' / 'ahdcode'), 'version'], check=True, capture_output=True, text=True).stdout.strip(), f'AhdCode v{self.version}')
            subprocess.run(['sh', str(custom_prefix / 'current' / 'install.sh'), '--prefix', str(custom_prefix), '--uninstall'], env=env, check=True, capture_output=True, text=True)
            self.assertFalse(custom_prefix.exists())

            # Uninstall default root and confirm user data + unrelated shell lines preserved.
            subprocess.run(['sh', str(ahd_root / 'current' / 'install.sh'), '--uninstall'], env=env, check=True, capture_output=True, text=True)
            self.assertFalse(ahd_root.exists())
            self.assertEqual(user_db.read_text().strip(), 'sentinel-data')
            zshrc_after = (fake_home / '.zshrc').read_text()
            self.assertNotIn('AhdCode PATH', zshrc_after)
            self.assertIn('# user-config-keep', zshrc_after)

    def _exercise_pkg_isolated(self, pkg_path, label):
        dominfo = subprocess.run(['installer', '-dominfo', '-pkg', str(pkg_path)], check=True, capture_output=True, text=True)
        self.assertIn('CurrentUserHomeDirectory', dominfo.stdout)
        with tempfile.TemporaryDirectory(prefix=f'ahdcode-pkg-{label}-') as tmp:
            work = pathlib.Path(tmp)
            expanded = work / 'expanded'
            subprocess.run(['pkgutil', '--expand-full', str(pkg_path), str(expanded)], check=True, capture_output=True, text=True)
            dist_xml = (expanded / 'Distribution').read_text()
            self.assertIn('enable_currentUserHome="true"', dist_xml)
            self.assertIn('enable_localSystem="false"', dist_xml)
            self.assertIn('hostArchitectures="arm64"', dist_xml)
            comp = next(expanded.glob('*.pkg'))
            pkg_info = ElementTree.parse(comp / 'PackageInfo').getroot()
            self.assertEqual(pkg_info.get('auth'), 'none')
            self.assertEqual(pkg_info.get('install-location').strip('/'), 'Library/AhdCode')

            fake_home = work / 'home'
            ahd_root, user_db = seed_isolated_home(fake_home)
            shutil.copytree(comp / 'Payload', ahd_root, symlinks=True, dirs_exist_ok=True)
            env = {'HOME': str(fake_home), 'PATH': os.environ['PATH']}
            postinstall = comp / 'Scripts' / 'postinstall'
            subprocess.run(['sh', str(postinstall), str(pkg_path), str(ahd_root), str(fake_home)], env=env, check=True, capture_output=True, text=True)
            # Idempotency check on second postinstall invocation.
            subprocess.run(['sh', str(postinstall), str(pkg_path), str(ahd_root), str(fake_home)], env=env, check=True, capture_output=True, text=True)

            self.assertEqual((ahd_root / 'versions' / '2.9.0' / 'sentinel.txt').read_text().strip(), 'keep-2.9.0')
            self.assertEqual(os.readlink(ahd_root / 'current'), f'versions/{self.version}')
            ver_dir = ahd_root / 'versions' / self.version
            self.assertTrue(os.access(ver_dir / 'bin' / 'ahdcode', os.X_OK))
            self.assertTrue(os.access(ver_dir / 'install.sh', os.X_OK))
            for helper in EXPECTED_HELPERS:
                self.assertTrue(os.access(ver_dir / 'libexec' / 'ahdcode' / helper, os.X_OK))
            for profile_name in ('.zprofile', '.zshrc', '.bash_profile'):
                self.assertEqual((fake_home / profile_name).read_text().count('# >>> AhdCode PATH >>>'), 1, profile_name)

            self.assertEqual(subprocess.run([str(ahd_root / 'bin' / 'ahdcode'), 'version'], check=True, capture_output=True, text=True).stdout.strip(), f'AhdCode v{self.version}')
            verify_sqlite_execution(ahd_root / 'bin' / 'ahdcode', work)

            # Uninstall preserves Library/Application Support and ~/.zshrc user lines.
            subprocess.run(['sh', str(ahd_root / 'current' / 'install.sh'), '--uninstall'], env=env, check=True, capture_output=True, text=True)
            self.assertFalse(ahd_root.exists())
            self.assertEqual(user_db.read_text().strip(), 'sentinel-data')
            self.assertIn('# user-config-keep', (fake_home / '.zshrc').read_text())

    def test_pkg_metadata_and_isolated_install_execute_uninstall(self):
        pkg, _ = self._artifact('arm64', 'pkg')
        self._exercise_pkg_isolated(pkg, 'standalone')

    def test_dmg_mount_contents_and_pkg_install(self):
        dmg, _ = self._artifact('arm64', 'dmg')
        standalone_pkg, _ = self._artifact('arm64', 'pkg')
        with tempfile.TemporaryDirectory(prefix='ahdcode-dmg-test-') as tmp:
            mount = pathlib.Path(tmp) / 'mnt'
            mount.mkdir()
            subprocess.run(
                ['hdiutil', 'attach', '-readonly', '-nobrowse', '-noautoopen', '-mountpoint', str(mount), str(dmg)],
                check=True, capture_output=True, text=True,
            )
            try:
                self.assertEqual((mount / 'VERSION').read_text().strip(), self.version)
                inst_txt = (mount / 'INSTALL.txt').read_text()
                self.assertIn(self.version, inst_txt)
                self.assertIn('--uninstall', inst_txt)
                self.assertIn('SHA-256', inst_txt)
                self.assertNotIn('spctl', inst_txt)
                self.assertNotIn('xattr -d', inst_txt)
                dmg_pkg = mount / standalone_pkg.name
                self.assertTrue(dmg_pkg.is_file())
                self.assertEqual(sha256_file(dmg_pkg), sha256_file(standalone_pkg))
                self._exercise_pkg_isolated(dmg_pkg, 'from-dmg')
            finally:
                subprocess.run(['hdiutil', 'detach', str(mount)], check=False, capture_output=True, text=True)

    def test_x64_packaging_artifacts_if_present(self):
        has_x64 = any(r['architecture'] == 'amd64' for r in self.manifest['artifacts'])
        if not has_x64:
            self.skipTest('no amd64 artifacts in dist dir')
        targz, _ = self._artifact('amd64', 'tar.gz')
        pkg, _ = self._artifact('amd64', 'pkg')
        dmg, _ = self._artifact('amd64', 'dmg')

        with tempfile.TemporaryDirectory(prefix='ahdcode-x64-check-') as tmp:
            work = pathlib.Path(tmp)
            ext = work / 'tgz'
            ext.mkdir()
            with tarfile.open(targz, 'r:gz') as tf:
                tf.extractall(ext, filter='data')
            root_dir = next(ext.iterdir())
            bins = [root_dir / 'payload' / 'bin' / 'ahdcode'] + [
                root_dir / 'payload' / 'libexec' / 'ahdcode' / h for h in EXPECTED_HELPERS
            ]
            for b in bins:
                self.assertTrue(os.access(b, os.X_OK), str(b))
                archs = subprocess.run(['lipo', '-archs', str(b)], check=True, capture_output=True, text=True).stdout.strip()
                self.assertEqual(archs, 'x86_64', f'{b} has archs {archs!r}')

            exp = work / 'pkg'
            subprocess.run(['pkgutil', '--expand-full', str(pkg), str(exp)], check=True, capture_output=True, text=True)
            dist_xml = (exp / 'Distribution').read_text()
            self.assertIn('hostArchitectures="x86_64"', dist_xml)
            self.assertIn('enable_currentUserHome="true"', dist_xml)
            self.assertIn('enable_localSystem="false"', dist_xml)
            comp = next(exp.glob('*.pkg'))
            self.assertEqual(ElementTree.parse(comp / 'PackageInfo').getroot().get('auth'), 'none')

            mnt = work / 'mnt'
            mnt.mkdir()
            subprocess.run(
                ['hdiutil', 'attach', '-readonly', '-nobrowse', '-noautoopen', '-mountpoint', str(mnt), str(dmg)],
                check=True, capture_output=True, text=True,
            )
            try:
                self.assertEqual((mnt / 'VERSION').read_text().strip(), self.version)
                self.assertTrue((mnt / pkg.name).is_file())
                self.assertEqual(sha256_file(mnt / pkg.name), sha256_file(pkg))
            finally:
                subprocess.run(['hdiutil', 'detach', str(mnt)], check=False, capture_output=True, text=True)


if __name__ == '__main__':
    unittest.main()
