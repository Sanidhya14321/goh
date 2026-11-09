import unittest
from stats import median

class Median(unittest.TestCase):
    def test_even(self):
        self.assertEqual(median([4, 1, 3, 2]), 2.5)
    def test_unsorted_odd(self):
        self.assertEqual(median([9, 1, 5]), 5)
    def test_empty(self):
        with self.assertRaises(ValueError):
            median([])

if __name__ == '__main__':
    unittest.main()
