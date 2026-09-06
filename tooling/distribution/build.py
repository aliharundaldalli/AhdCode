#!/usr/bin/env python3
"""Build self-contained release artifacts. Native builds run strictly sequentially."""
import argparse,hashlib,json,os,pathlib,shutil,subprocess,tarfile,time,zipfile
R=pathlib.Path(__file__).resolve().parents[2]
def sha(p):
 h=hashlib.sha256()
 with open(p,'rb') as f:
  for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
 return h.hexdigest()
def run(args,**kwargs):subprocess.run([str(a) for a in args],check=True,**kwargs)
def extract(archive,dest):
 if str(archive).endswith('.zip'):
  with zipfile.ZipFile(archive) as z:z.extractall(dest)
 else:
  with tarfile.open(archive) as t:t.extractall(dest,filter='data')
def build(target,output,goos,arch,ldflags='-buildid='):
 env=dict(os.environ,GOFLAGS='-p=1',GOMAXPROCS='2',GOOS=goos,GOARCH=arch,CGO_ENABLED='0')
 print('Building',goos,arch,target,flush=True)
 run(['nice','-n','10','go','build','-trimpath','-ldflags='+ldflags,'-o',output,target],cwd=R,env=env)
 time.sleep(4)
def licenses(payload,modules):
 dest=payload/'licenses';dest.mkdir()
 for name in ['LICENSE','THIRD_PARTY_NOTICES_MYSQL.md','THIRD_PARTY_NOTICES_NUMERIC.md','THIRD_PARTY_NOTICES_PLOT.md','THIRD_PARTY_NOTICES_SQLITE.md']:shutil.copy2(R/name,payload/name)
 shutil.copy2(R/'tooling/distribution/THIRD_PARTY_NOTICES.md',payload/'THIRD_PARTY_NOTICES.md')
 shutil.copytree(R/'internal/backend/golang/ahdruntime/mysqlvendor/vendor',dest/'mysql-source')
 (dest/'bootstrap').mkdir();shutil.copy2(R/'internal/initweb/templates/vendor/bootstrap/LICENSE',dest/'bootstrap/LICENSE')
 shutil.copy2(R/'tooling/latex/resources.json',dest/'latex-resources.json')
 inventory=[]
 for mod in modules:
  if mod.get('Main'):continue
  path=pathlib.Path(mod.get('Dir',''))
  if not path.is_dir() or not mod.get('Dir'):continue
  entry={'module':mod['Path'],'version':mod.get('Version'),'source':'https://pkg.go.dev/'+mod['Path']+'@'+mod.get('Version',''),'notices':[]}
  for file in path.iterdir():
   if file.is_file() and (file.name.upper().startswith(('LICENSE','COPYING','NOTICE','PATENTS'))):
    rel=pathlib.Path('modules')/(mod['Path'].replace('/','_')+'@'+mod.get('Version',''))/file.name
    (dest/rel).parent.mkdir(parents=True,exist_ok=True);shutil.copy2(file,dest/rel);entry['notices'].append(str(rel))
  if not entry['notices']:raise RuntimeError('Missing license: '+mod['Path'])
  inventory.append(entry)
 (dest/'modules.json').write_text(json.dumps(inventory,indent=2)+'\n')
def zip_payload(payload,out):
 with zipfile.ZipFile(out,'w',zipfile.ZIP_DEFLATED,compresslevel=6) as z:
  for p in sorted(payload.rglob('*')):
   if p.is_file():
    info=zipfile.ZipInfo(p.relative_to(payload).as_posix(),(2026,1,1,0,0,0));info.external_attr=(p.stat().st_mode&0o777)<<16;info.compress_type=zipfile.ZIP_DEFLATED;z.writestr(info,p.read_bytes())

# The LaTeX runtime is staged outside the repository, so it is the one payload
# component a release could plausibly acquire in a damaged form: a truncated
# copy, a text-mode transfer, or a Git LFS pointer standing in for the real
# file. Existence is not evidence, so the bundle is checked by magic, size and
# digest, and each engine by its executable format for the platform it ships to.
LFS_POINTER = b'version https://git-lfs.github.com/spec/'
TTB_MAGIC = b'tectonicbundle'
EXECUTABLE_MAGIC = {'darwin': [b'\xcf\xfa\xed\xfe', b'\xca\xfe\xba\xbe'], 'linux': [b'\x7fELF'], 'windows': [b'MZ']}

def check_bundle(path, expected):
 if not path.is_file(): raise SystemExit('LaTeX bundle is missing: ' + str(path))
 head = path.open('rb').read(len(LFS_POINTER))
 if head.startswith(LFS_POINTER): raise SystemExit('LaTeX bundle is a Git LFS pointer, not the bundle: ' + str(path))
 if not head.startswith(TTB_MAGIC): raise SystemExit('LaTeX bundle does not start with the Tectonic bundle magic: ' + str(path))
 size = path.stat().st_size
 if size != expected['ttb_size']: raise SystemExit('LaTeX bundle is %d bytes, expected %d: %s' % (size, expected['ttb_size'], path))
 digest = sha(path)
 if digest != expected['ttb_sha256']: raise SystemExit('LaTeX bundle digest %s does not match the pinned %s' % (digest, expected['ttb_sha256']))

def check_engine(path, goos):
 if not path.is_file(): raise SystemExit('LaTeX engine is missing: ' + str(path))
 head = path.open('rb').read(4)
 if head.startswith(LFS_POINTER[:4]): raise SystemExit('LaTeX engine is a Git LFS pointer: ' + str(path))
 if not any(head.startswith(magic) for magic in EXECUTABLE_MAGIC[goos]):
  raise SystemExit('LaTeX engine is not a %s executable: %s' % (goos, path))
 if path.stat().st_size < 1 << 20: raise SystemExit('LaTeX engine is implausibly small: ' + str(path))


def check_vsix(path):
 """The editor extension is built by a separate, network-using step, so the
 platform build treats it as an input to verify rather than trust."""
 if not path.is_file(): raise SystemExit('VS Code extension package is missing: ' + str(path))
 if not zipfile.is_zipfile(path): raise SystemExit('VS Code extension package is not a .vsix archive: ' + str(path))
 with zipfile.ZipFile(path) as archive:
  names=set(archive.namelist())
  for required in ['extension.vsixmanifest','extension/package.json','extension/extension.js']:
   if required not in names: raise SystemExit('VS Code extension package is missing %s: %s' % (required,path))
  manifest=json.loads(archive.read('extension/package.json'))
  if not any(n.startswith('extension/node_modules/vscode-languageclient/') for n in names):
   raise SystemExit('VS Code extension package does not bundle its language client: ' + str(path))
 return manifest['version']


def zip_tree(source, out, root):
 """Archive a staging directory under one top-level folder, with the execute
 bits intact after Finder expands it.

 A zip written by Python records mode 0755 correctly and Info-ZIP's `unzip`
 honours it, but Finder expands archives with `ditto`, which drops the bit and
 leaves Install.command and every binary non-executable. `ditto -c -k` writes
 the metadata ditto itself reads back, so the archive survives a double-click."""
 renamed=source.parent/root
 source.rename(renamed)
 try:
  run(['ditto','-c','-k','--sequesterRsrc','--keepParent',renamed,out])
 finally:
  renamed.rename(source)
 verify_zip_modes(out,root)

def verify_zip_modes(archive, root):
 """Expand the way Finder does and insist the entry points are runnable."""
 import tempfile
 with tempfile.TemporaryDirectory(prefix='ahdcode-zipcheck-') as workspace:
  run(['ditto','-x','-k',archive,workspace])
  for relative in ['Install.command','install.sh','payload/bin/ahdcode']:
   path=pathlib.Path(workspace)/root/relative
   if not path.is_file(): raise SystemExit('the archive is missing ' + relative)
   if not os.access(path,os.X_OK): raise SystemExit('Finder would expand ' + relative + ' without its execute bit')


MACHO_MAGIC = {b'\xcf\xfa\xed\xfe', b'\xce\xfa\xed\xfe', b'\xca\xfe\xba\xbe'}

def macho_files(root):
 """Every Mach-O file in the payload. Notarization refuses a package that
 carries even one unsigned executable, so this looks at magic numbers rather
 than trusting file names or the execute bit."""
 found=[]
 for p in sorted(root.rglob('*')):
  if not p.is_file() or p.is_symlink(): continue
  with open(p,'rb') as f: head=f.read(4)
  if head in MACHO_MAGIC: found.append(p)
 return found

def sign_payload(root, identity):
 """Sign with the hardened runtime and a secure timestamp, which notarization
 requires. The private Go toolchain is signed too: it ships inside the package
 and Apple checks all of it."""
 targets=macho_files(root)
 if not targets: raise SystemExit('no Mach-O files found to sign under ' + str(root))
 for target in targets:
  run(['codesign','--force','--options','runtime','--timestamp','--sign',identity,target])
 for target in targets:
  result=subprocess.run(['codesign','--verify','--strict','--verbose=1',str(target)],capture_output=True,text=True)
  if result.returncode!=0: raise SystemExit('signature did not verify: %s\n%s' % (target,result.stderr.strip()))
 print('Signed and verified',len(targets),'Mach-O files',flush=True)

def main():
 ap=argparse.ArgumentParser();ap.add_argument('--downloads',type=pathlib.Path,required=True);ap.add_argument('--latex-runtime',type=pathlib.Path,required=True);ap.add_argument('--modules',type=pathlib.Path,required=True);ap.add_argument('--output',type=pathlib.Path,required=True);ap.add_argument('--sign-app',default='',help='Developer ID Application identity for the packaged binaries');ap.add_argument('--vsix',type=pathlib.Path,help='the .vsix from tooling/distribution/build_vsix.py');ap.add_argument('--without-vsix',action='store_true',help='deliberately ship no editor extension');ap.add_argument('--platform',choices=['darwin','windows','linux','all'],default='all');a=ap.parse_args()
 version=__import__('re').search(r'Number\s*=\s*"([^"]+)"',(R/'internal/ahdversion/version.go').read_text())[1]
 commit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=R,text=True).strip()
 raw=a.modules.read_text();decoder=json.JSONDecoder();modules=[]
 while raw.strip():m,end=decoder.raw_decode(raw.lstrip());modules.append(m);raw=raw.lstrip()[end:]
 go=json.loads((R/'tooling/distribution/go-assets.json').read_text());latex=json.loads((R/'tooling/latex/assets.json').read_text())
 # Omitting the extension is a decision, never an oversight: the release
 # that shipped without one is the reason this is not merely optional.
 if not a.vsix and not a.without_vsix: raise SystemExit('pass --vsix <file> (build it with tooling/distribution/build_vsix.py) or --without-vsix')
 vsix_version=check_vsix(a.vsix) if a.vsix else None
 bundle=a.latex_runtime/'ahdcode-latex.ttb';check_bundle(bundle,latex['bundle'])
 a.output.mkdir(parents=True,exist_ok=True);records=[]
 for goos,arch,label in [('darwin','arm64','macos-arm64'),('windows','amd64','windows-x64'),('linux','amd64','linux-x64')]:
  if a.platform not in ['all',goos]:continue
  extra=None
  staging=a.output/('stage-'+label);staging.mkdir();payload=staging/'payload';(payload/'bin').mkdir(parents=True);(payload/'libexec/ahdcode').mkdir(parents=True)
  asset=next(x for x in go['files'] if x['os']==goos and x['arch']==arch);archive=a.downloads/asset['filename'];assert sha(archive)==asset['sha256'];extract(archive,payload/'libexec')
  engine=next(x for x in latex['engines'] if x['goos']==goos and x['goarch']==arch);archive=a.downloads/engine['filename'];assert sha(archive)==engine['sha256'];latexdir=payload/'libexec/ahdcode/latex';latexdir.mkdir();extract(archive,latexdir)
  shutil.copy2(bundle,latexdir/bundle.name);shutil.copy2(R/'tooling/latex/THIRD_PARTY_NOTICES.txt',latexdir/'THIRD_PARTY_NOTICES.txt');shutil.copytree(R/'tooling/latex/licenses',latexdir/'licenses')
  # Re-check the staged copies: a transform between the source and the payload
  # is exactly what this gate exists to catch.
  check_bundle(latexdir/bundle.name,latex['bundle']);check_engine(latexdir/('tectonic.exe' if goos=='windows' else 'tectonic'),goos)
  suffix='.exe' if goos=='windows' else ''
  for name in ['ahdcode','ahdsqlite','ahdnumeric','ahdplot']:
   target=payload/('bin' if name=='ahdcode' else 'libexec/ahdcode')/(name+suffix);build('./cmd/'+name,target,goos,arch)
  if goos=='windows':
   # The stable launcher lives beside the release it activates; setup copies it
   # to <root>\bin\ahdcode.exe, the one directory that goes on PATH.
   (payload/'launcher').mkdir();build('./tooling/distribution/windows/launcher',payload/'launcher/ahdcode.exe',goos,arch)
  licenses(payload,modules)
  shutil.copytree(R/'internal/initweb/docbundle',payload/'docs')
  if a.vsix:
   editor=payload/'vscode';editor.mkdir();shutil.copy2(a.vsix,editor/a.vsix.name)
   (editor/'README.txt').write_text(
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
    'no network, and no separate download.\n')
  # Signing has to happen before the inventory is taken: codesign rewrites
  # each binary, so hashes recorded earlier would no longer match.
  if goos=='darwin' and a.sign_app: sign_payload(payload,a.sign_app)
  (payload/'VERSION').write_text(version+'\n');(staging/'VERSION').write_text(version+'\n')
  (payload/'BUILD.json').write_text(json.dumps({'version':version,'commit':commit,'platform':goos,'architecture':arch,'go':go['version'],'tectonic':latex['tectonic_version']},indent=2)+'\n')
  if goos=='windows':shutil.copy2(R/'tooling/distribution/windows/uninstall.ps1',payload/'uninstall.ps1')
  hashes={p.relative_to(payload).as_posix():sha(p) for p in sorted(payload.rglob('*')) if p.is_file()};(payload/'FILES.json').write_text(json.dumps(hashes,indent=2)+'\n')
  artifact=a.output/('AhdCode-'+version+'-'+label+('.exe' if goos=='windows' else '.dmg' if goos=='darwin' else '.tar.gz'))
  if goos=='windows':
   zipped=R/'tooling/distribution/windows/payload.zip'
   # -H=windowsgui keeps Explorer from opening a console: setup is graphical and
   # never reads stdin. It still attaches to a parent console when one exists.
   try:zip_payload(payload,zipped);build('./tooling/distribution/windows',artifact,goos,arch,'-buildid= -H=windowsgui')
   finally:zipped.unlink(missing_ok=True)
  else:
   shutil.copy2(R/'tooling/distribution/install.sh',staging/'install.sh')
   if goos=='darwin':
    shutil.copy2(R/'tooling/distribution/Install.command',staging/'Install.command')
    run(['hdiutil','create','-volname','AhdCode '+version,'-srcfolder',staging,'-format','UDZO','-ov',artifact])
    # The ZIP is the same staging tree as the DMG, so both carry one identity.
    extra=a.output/('AhdCode-'+version+'-'+label+'.zip');zip_tree(staging,extra,'AhdCode-'+version)
   else:
    with tarfile.open(artifact,'w:gz') as t:t.add(staging,arcname='AhdCode-'+version)
  records.append({'filename':artifact.name,'size':artifact.stat().st_size,'sha256':sha(artifact),'platform':goos,'architecture':arch,'components':['CLI','Studio (embedded)','starters (embedded)','English docs','Go '+go['version'],'ahdsqlite','ahdnumeric','ahdplot','Tectonic '+latex['tectonic_version']+' offline']+(['VS Code extension '+vsix_version] if vsix_version else [])+(['graphical per-user setup','stable ahdcode.exe launcher'] if goos=='windows' else []),'notices':'payload/THIRD_PARTY_NOTICES.md'})
  print('ARTIFACT',artifact,records[-1]['sha256'],flush=True)
  if extra is not None:
   companion=dict(records[-1]);companion['filename']=extra.name;companion['size']=extra.stat().st_size;companion['sha256']=sha(extra)
   records.append(companion);print('ARTIFACT',extra,companion['sha256'],flush=True)
  (a.output/('manifest-'+goos+'.json')).write_text(json.dumps({'version':version,'commit':commit,'artifacts':[r for r in records if r['platform']==goos]},indent=2)+'\n')
 # The standalone artifact and the copy inside each package are the same
 # bytes, because both are copies of the one file that was verified.
 if a.vsix: shutil.copy2(a.vsix,a.output/a.vsix.name)
 (a.output/'release-manifest.json').write_text(json.dumps({'version':version,'commit':commit,'extension':vsix_version,'artifacts':records},indent=2)+'\n')
if __name__=='__main__':main()
