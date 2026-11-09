import json

def load_settings(path, env=None):
    with open(path) as f:
        data = json.load(f)
    return {'port': data.get('port', 8000), 'debug': bool(data.get('debug', False)), 'name': data.get('name', 'app')}
