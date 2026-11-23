#!/usr/bin/env python3
"""Build GoHarness's architecture graph: packages, files and how they connect.

Reads `go list -json ./...` and the non-test Go files, and writes a graph
store (JSON) of nodes and edges:

  package nodes  name, layer, purpose (package doc), files, lines
  file nodes     package, purpose (leading doc comment), main declarations,
                 lines, internal packages it uses
  edges          package -> package imports (with the files that use them),
                 package -> file membership

Usage: python3 scripts/archgraph/archgraph.py [--go PATH] [--out FILE]
Run from the repository root. Only the code is read; nothing is executed.
"""
import argparse, json, os, re, subprocess, sys

MODULE = 'github.com/Sanidhya14321/goh/'
LAYERS = [
    ('entry', ['cmd/']),
    ('interface', ['internal/tui', 'internal/acp', 'internal/localapi', 'internal/remote']),
    ('orchestration', ['internal/runtime']),
    ('capabilities', ['internal/tools', 'internal/webfetch', 'internal/websearch', 'internal/lspclient', 'internal/mcpclient',
                      'internal/memory', 'internal/skills', 'internal/commands', 'internal/formatters', 'internal/structured']),
    ('models', ['internal/provider']),
    ('persistence', ['internal/session', 'internal/snapshots', 'internal/projectsnap', 'internal/rewind']),
    ('foundation', ['internal/core', 'internal/platform', 'internal/sandbox', 'internal/secretstore', 'internal/ignore', 'internal/testutil']),
]

def layer(path):
    for name, prefixes in LAYERS:
        if any(path == p or path.startswith(p) for p in prefixes):
            return name
    return 'other'

def go_list(go):
    raw = subprocess.run([go, 'list', '-json', './...'], capture_output=True, text=True, check=True).stdout
    dec, i, out = json.JSONDecoder(), 0, []
    while True:
        rest = raw[i:].lstrip()
        if not rest:
            return out
        obj, n = dec.raw_decode(rest)
        out.append(obj)
        i = len(raw) - len(rest) + n

def first_sentence(text, limit=220):
    text = ' '.join(text.split())
    m = re.match(r'(.+?[.!?])(\s|$)', text)
    s = m.group(1) if m else text
    return s if len(s) <= limit else s[:limit - 1] + '…'

DECL = re.compile(r'^(func|type|var|const)\s*(\([^)]*\)\s*)?([A-Za-z_]\w*)?')

def file_facts(path):
    lines = open(path, encoding='utf-8').read().splitlines()
    package_doc, purpose, comment, decls, imports = '', '', [], [], set()
    in_import = False
    for line in lines:
        s = line.strip()
        if s.startswith('import ('):
            in_import = True
            continue
        if in_import:
            if s == ')':
                in_import = False
            else:
                m = re.search(r'"(' + re.escape(MODULE) + r'[^"]+)"', s)
                if m:
                    imports.add(m.group(1).replace(MODULE, ''))
            continue
        m = re.match(r'import\s+(\w+\s+)?"(' + re.escape(MODULE) + r'[^"]+)"', s)
        if m:
            imports.add(m.group(2).replace(MODULE, ''))
            continue
        if line.startswith('//'):
            comment.append(line[2:].strip())
            continue
        if line.startswith('package '):
            if comment and comment[0].startswith('Package '):
                package_doc = first_sentence(' '.join(comment), 300)
            comment = []
            continue
        d = DECL.match(line)
        if d:
            name = d.group(3) or ''
            if comment and not purpose and not comment[0].startswith(('go:', 'nolint')):
                purpose = first_sentence(' '.join(comment))
            if name and len(decls) < 8 and not name.startswith('_'):
                kind = 'method' if d.group(2) else d.group(1)
                decls.append(f'{kind} {name}')
        if not line.startswith('//'):
            comment = [] if s else comment
    if not purpose and package_doc:
        purpose = package_doc
    return {'purpose': purpose, 'package_doc': package_doc, 'decls': decls, 'lines': len(lines), 'uses': sorted(imports)}

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--go', default=os.environ.get('GO', 'go'))
    ap.add_argument('--out', default='docs/architecture/graph.json')
    ap.add_argument('--purposes', default='docs/architecture/purposes.json')
    args = ap.parse_args()
    curated = json.load(open(args.purposes)) if os.path.exists(args.purposes) else {}
    nodes, edges = [], []
    for p in sorted(go_list(args.go), key=lambda p: p['ImportPath']):
        path = p['ImportPath'].replace(MODULE, '')
        files, package_doc = [], ''
        for f in sorted(p.get('GoFiles', [])):
            facts = file_facts(os.path.join(p['Dir'], f))
            package_doc = package_doc or facts.pop('package_doc')
            facts.pop('package_doc', None)
            fid = f'{path}/{f}'
            if fid in curated:
                facts['purpose'], facts['curated'] = curated[fid], True
            files.append(fid)
            nodes.append({'id': fid, 'kind': 'file', 'package': path, 'name': f, **facts})
            edges.append({'from': path, 'to': fid, 'kind': 'contains'})
        internal = sorted(d.replace(MODULE, '') for d in p.get('Imports', []) if d.startswith(MODULE))
        for dep in internal:
            users = [n['name'] for n in nodes if n['kind'] == 'file' and n['package'] == path and dep in n['uses']]
            edges.append({'from': path, 'to': dep, 'kind': 'imports', 'files': users})
        nodes.append({'id': path, 'kind': 'package', 'layer': layer(path), 'purpose': curated.get(path) or package_doc,
                      'files': files, 'lines': sum(n['lines'] for n in nodes if n['kind'] == 'file' and n['package'] == path)})
    pkgs = {n['id'] for n in nodes if n['kind'] == 'package'}
    for n in nodes:
        if n['kind'] == 'package':
            n['fan_out'] = sum(1 for e in edges if e['kind'] == 'imports' and e['from'] == n['id'])
            n['fan_in'] = sum(1 for e in edges if e['kind'] == 'imports' and e['to'] == n['id'])
    files = [n['id'] for n in nodes if n['kind'] == 'file']
    known = set(files) | pkgs
    missing = [f for f in sorted(known) if f not in curated]
    stale = [f for f in curated if f not in known]
    if missing:
        print('no curated purpose (add to ' + args.purposes + '): ' + ', '.join(missing), file=sys.stderr)
    if stale:
        print('curated purposes for files that no longer exist: ' + ', '.join(stale), file=sys.stderr)
    os.makedirs(os.path.dirname(args.out), exist_ok=True)
    with open(args.out, 'w') as fh:
        json.dump({'module': MODULE.rstrip('/'), 'layers': [l for l, _ in LAYERS], 'nodes': nodes, 'edges': edges}, fh, indent=1)
    print(f'{args.out}: {len(pkgs)} packages, {sum(1 for n in nodes if n["kind"] == "file")} files, '
          f'{sum(1 for e in edges if e["kind"] == "imports")} import edges')

if __name__ == '__main__':
    sys.exit(main())
