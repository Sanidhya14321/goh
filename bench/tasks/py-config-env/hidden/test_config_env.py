import unittest
from app.config import load_settings
from app import server

class Env(unittest.TestCase):
    def test_file_only(self):
        self.assertEqual(load_settings('settings.json', env={}), {'port': 8080, 'debug': False, 'name': 'shop'})
    def test_overrides(self):
        s = load_settings('settings.json', env={'APP_PORT': '9000', 'APP_DEBUG': 'TRUE', 'APP_NAME': 'api'})
        self.assertEqual(s, {'port': 9000, 'debug': True, 'name': 'api'})
    def test_debug_false_strings(self):
        for v in ('false', 'False', '0'):
            self.assertFalse(load_settings('settings.json', env={'APP_DEBUG': v})['debug'])
    def test_server_uses_loader(self):
        import os
        os.environ['APP_PORT'] = '7000'
        try:
            self.assertIn(':7000', server.describe('settings.json'))
        finally:
            del os.environ['APP_PORT']

if __name__ == '__main__':
    unittest.main()
