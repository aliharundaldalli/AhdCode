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
def build(target,output,goos,arch):
 env=dict(os.environ,GOFLAGS='-p=1',GOMAXPROCS='2',GOOS=goos,GOARCH=arch,CGO_ENABLED='0')
 print('Building',goos,arch,target,flush=True)
 run(['nice','-n','10','go','build','-trimpath','-ldflags=-buildid=','-o',output,target],cwd=R,env=env)
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
def main():
 ap=argparse.ArgumentParser();ap.add_argument('--downloads',type=pathlib.Path,required=True);ap.add_argument('--latex-runtime',type=pathlib.Path,required=True);ap.add_argument('--modules',type=pathlib.Path,required=True);ap.add_argument('--output',type=pathlib.Path,required=True);ap.add_argument('--platform',choices=['darwin','windows','linux','all'],default='all');a=ap.parse_args()
 version=__import__('re').search(r'Number\s*=\s*"([^"]+)"',(R/'internal/ahdversion/version.go').read_text())[1]
 commit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=R,text=True).strip()
 raw=a.modules.read_text();decoder=json.JSONDecoder();modules=[]
 while raw.strip():m,end=decoder.raw_decode(raw.lstrip());modules.append(m);raw=raw.lstrip()[end:]
 go=json.loads((R/'tooling/distribution/go-assets.json').read_text());latex=json.loads((R/'tooling/latex/assets.json').read_text())
 bundle=a.latex_runtime/'ahdcode-latex.ttb';assert sha(bundle)==latex['bundle']['ttb_sha256']
 a.output.mkdir(parents=True,exist_ok=True);records=[]
 for goos,arch,label in [('darwin','arm64','macos-arm64'),('windows','amd64','windows-x64'),('linux','amd64','linux-x64')]:
  if a.platform not in ['all',goos]:continue
  staging=a.output/('stage-'+label);staging.mkdir();payload=staging/'payload';(payload/'bin').mkdir(parents=True);(payload/'libexec/ahdcode').mkdir(parents=True)
  asset=next(x for x in go['files'] if x['os']==goos and x['arch']==arch);archive=a.downloads/asset['filename'];assert sha(archive)==asset['sha256'];extract(archive,payload/'libexec')
  engine=next(x for x in latex['engines'] if x['goos']==goos and x['goarch']==arch);archive=a.downloads/engine['filename'];assert sha(archive)==engine['sha256'];latexdir=payload/'libexec/ahdcode/latex';latexdir.mkdir();extract(archive,latexdir)
  shutil.copy2(bundle,latexdir/bundle.name);shutil.copy2(R/'tooling/latex/THIRD_PARTY_NOTICES.txt',latexdir/'THIRD_PARTY_NOTICES.txt');shutil.copytree(R/'tooling/latex/licenses',latexdir/'licenses')
  suffix='.exe' if goos=='windows' else ''
  for name in ['ahdcode','ahdsqlite','ahdnumeric','ahdplot']:
   target=payload/('bin' if name=='ahdcode' else 'libexec/ahdcode')/(name+suffix);build('./cmd/'+name,target,goos,arch)
  licenses(payload,modules)
  shutil.copytree(R/'internal/initweb/docbundle',payload/'docs')
  (payload/'VERSION').write_text(version+'\n');(staging/'VERSION').write_text(version+'\n')
  (payload/'BUILD.json').write_text(json.dumps({'version':version,'commit':commit,'platform':goos,'architecture':arch,'go':go['version'],'tectonic':latex['tectonic_version']},indent=2)+'\n')
  if goos=='windows':shutil.copy2(R/'tooling/distribution/windows/uninstall.ps1',payload/'uninstall.ps1')
  hashes={p.relative_to(payload).as_posix():sha(p) for p in sorted(payload.rglob('*')) if p.is_file()};(payload/'FILES.json').write_text(json.dumps(hashes,indent=2)+'\n')
  artifact=a.output/('AhdCode-'+version+'-'+label+('.exe' if goos=='windows' else '.dmg' if goos=='darwin' else '.tar.gz'))
  if goos=='windows':
   zipped=R/'tooling/distribution/windows/payload.zip'
   try:zip_payload(payload,zipped);build('./tooling/distribution/windows',artifact,goos,arch)
   finally:zipped.unlink(missing_ok=True)
  else:
   shutil.copy2(R/'tooling/distribution/install.sh',staging/'install.sh')
   if goos=='darwin':
    shutil.copy2(R/'tooling/distribution/Install.command',staging/'Install.command')
    run(['hdiutil','create','-volname','AhdCode '+version,'-srcfolder',staging,'-format','UDZO','-ov',artifact])
   else:
    with tarfile.open(artifact,'w:gz') as t:t.add(staging,arcname='AhdCode-'+version)
  records.append({'filename':artifact.name,'size':artifact.stat().st_size,'sha256':sha(artifact),'platform':goos,'architecture':arch,'components':['CLI','Studio (embedded)','starters (embedded)','English docs','Go '+go['version'],'ahdsqlite','ahdnumeric','ahdplot','Tectonic '+latex['tectonic_version']+' offline'],'notices':'payload/THIRD_PARTY_NOTICES.md'})
  (a.output/('manifest-'+goos+'.json')).write_text(json.dumps({'version':version,'commit':commit,'artifacts':[records[-1]]},indent=2)+'\n');print('ARTIFACT',artifact,records[-1]['sha256'],flush=True)
 (a.output/'release-manifest.json').write_text(json.dumps({'version':version,'commit':commit,'artifacts':records},indent=2)+'\n')
if __name__=='__main__':main()
