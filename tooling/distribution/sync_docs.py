#!/usr/bin/env python3
"""Reconcile the offline English project bundle from the canonical release documents.

The bundle is the exact-version English snapshot copied into every generated Web
project. Turkish translations, repository-only examples, and images are excluded,
and cross-document links are rewritten to the flat bundle layout. This script only
rewrites `internal/initweb/docbundle/` and the generated manifest in
`internal/initweb/docbundle.go`; it never edits other program logic.
"""
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
BUNDLE = ROOT / 'internal/initweb/docbundle'
MANIFEST_FILE = ROOT / 'internal/initweb/docbundle.go'


def sources():
    files = sorted(p for p in (ROOT / 'docs').glob('*.md') if not p.name.endswith('_TR.md'))
    return files + [ROOT / 'README.md', ROOT / 'AHDCODE_LANGUAGE_SPEC_v0.1.md']


def rewrite_links(document, text, included):
    def replace(match):
        label, target = match.groups()
        if re.match(r'\w+://|mailto:|#', target):
            return match[0]
        path, separator, anchor = target.partition('#')
        resolved = (document.parent / path).resolve()
        if resolved in included:
            return '[' + label + '](' + included[resolved] + (separator + anchor if separator else '') + ')'
        # Repository-only paths and Turkish translations are not shipped, so the
        # link text stays as plain prose rather than becoming a broken link.
        return label
    return re.sub(r'\[([^\]\n]+)\]\(([^)\s]+)\)', replace, text)


def main():
    documents = sources()
    included = {p.resolve(): p.name for p in documents}
    for stale in BUNDLE.glob('*.md'):
        if stale.name not in included.values():
            stale.unlink()
    for document in documents:
        text = rewrite_links(document, document.read_text(), included)
        text = re.sub(r'<img[^>]+>', '', text)
        (BUNDLE / document.name).write_text(text)
    manifest = ('var documentationManifest = []string{\n'
                + ''.join('\t"' + p.name + '",\n' for p in documents) + '}')
    source = MANIFEST_FILE.read_text()
    updated = re.sub(r'var documentationManifest = \[\]string\{.*?\n\}', lambda _: manifest, source, flags=re.S)
    if updated == source and manifest not in source:
        raise SystemExit('documentationManifest declaration not found in ' + str(MANIFEST_FILE))
    MANIFEST_FILE.write_text(updated)
    print('Synchronized', len(documents), 'English documents')


if __name__ == '__main__':
    main()
