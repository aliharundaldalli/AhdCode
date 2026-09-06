#!/usr/bin/env python3
"""Reconcile the offline English bundle from canonical release documents."""
from pathlib import Path
import re
root=Path(__file__).resolve().parents[2]
source=sorted(p for p in (root/'docs').glob('*.md') if not p.name.endswith('_TR.md'))
source += [root/'README.md',root/'AHDCODE_LANGUAGE_SPEC_v0.1.md']
names={p.resolve():p.name for p in source}
dest=root/'internal/initweb/docbundle'
for p in source:
 text=p.read_text()
 def link(m):
  label,target=m.groups()
  if re.match(r'\w+://|mailto:|#',target):return m[0]
  path,sep,anchor=target.partition('#');resolved=(p.parent/path).resolve()
  if resolved in names:return '['+label+']('+names[resolved]+(sep+anchor if sep else '')+')'
  # Repository-only examples and Turkish translations are not falsely linked offline.
  return label
 text=re.sub(r'\[([^\]\n]+)\]\(([^)\s]+)\)',link,text)
 text=re.sub(r'<img[^>]+>','',text)
 (dest/p.name).write_text(text)
manifest='var documentationManifest = []string{\n'+''.join('\t"'+p.name+'",\n' for p in source)+'}'
p=root/'internal/initweb/docbundle.go';text=p.read_text();text=re.sub(r'var documentationManifest = \[\]string\{.*?\n\}',lambda _:manifest,text,flags=re.S);text=text.replace('content: content,','content: append([]byte("<!-- Exact-version documentation: "+ahdversion.Display+" -->\\n\\n"), content...),');text=text.replace('list, view, create, and delete users.','list, view, create, edit names, and delete members after confirmation.');p.write_text(text)
print('Synchronized',len(source),'English documents')
