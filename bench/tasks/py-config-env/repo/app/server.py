import json

def describe(path):
    with open(path) as f:
        s = json.load(f)
    return '%s on :%d%s' % (s.get('name', 'app'), s.get('port', 8000), ' (debug)' if s.get('debug') else '')
