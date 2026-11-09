import unittest
from text_utils import slugify

class Slug(unittest.TestCase):
    def test_basic(self):
        self.assertEqual(slugify('Hello, World!'), 'hello-world')
    def test_accents(self):
        self.assertEqual(slugify('Café Crème'), 'cafe-creme')
    def test_runs_and_edges(self):
        self.assertEqual(slugify('  --A__b  c--  '), 'a-b-c')
    def test_empty(self):
        self.assertEqual(slugify('!!!'), '')

if __name__ == '__main__':
    unittest.main()
