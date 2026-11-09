from app.config import load_settings

def describe(path):
    s = load_settings(path)
    return '%s on :%d%s' % (s['name'], s['port'], ' (debug)' if s['debug'] else '')
