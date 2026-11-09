import unittest
from wrapping import wrap

class W(unittest.TestCase):
    def test_basic(self):
        self.assertEqual(wrap('the quick brown fox', 10), ['the quick', 'brown fox'])
    def test_long_word(self):
        self.assertEqual(wrap('abcdefghij kl', 4), ['abcd', 'efgh', 'ij', 'kl'])
    def test_spaces(self):
        self.assertEqual(wrap('  a    b  ', 3), ['a b'])
    def test_empty(self):
        self.assertEqual(wrap('', 5), [])
    def test_limit(self):
        for line in wrap('lorem ipsum dolor sit amet consectetur adipiscing', 7):
            self.assertLessEqual(len(line), 7)

if __name__ == '__main__':
    unittest.main()
