import json
import os

_TRUE = {'true', '1', 'yes'}

def load_settings(path, env=None):
    env = os.environ if env is None else env
    with open(path) as f:
        data = json.load(f)
    s = {'port': int(data.get('port', 8000)), 'debug': bool(data.get('debug', False)), 'name': data.get('name', 'app')}
    if 'APP_PORT' in env:
        s['port'] = int(env['APP_PORT'])
    if 'APP_DEBUG' in env:
        s['debug'] = env['APP_DEBUG'].strip().lower() in _TRUE
    if 'APP_NAME' in env:
        s['name'] = env['APP_NAME']
    return s
