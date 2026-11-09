import unittest
from pages import paginate, page_count

class P(unittest.TestCase):
    def test_first(self):
        self.assertEqual(paginate(list(range(10)), 1, 4), [0, 1, 2, 3])
    def test_last_short(self):
        self.assertEqual(paginate(list(range(10)), 3, 4), [8, 9])
    def test_past_end(self):
        self.assertEqual(paginate(list(range(10)), 4, 4), [])
    def test_count(self):
        self.assertEqual(page_count(list(range(10)), 4), 3)
        self.assertEqual(page_count([], 4), 0)

if __name__ == '__main__':
    unittest.main()
