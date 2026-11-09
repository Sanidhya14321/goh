import unittest
from intervals import merge

class M(unittest.TestCase):
    def test_unsorted(self):
        self.assertEqual(merge([[8, 10], [1, 3], [2, 6]]), [[1, 6], [8, 10]])
    def test_touching(self):
        self.assertEqual(merge([[1, 3], [3, 5]]), [[1, 5]])
    def test_empty(self):
        self.assertEqual(merge([]), [])
    def test_input_untouched(self):
        data = [[1, 4], [2, 3]]
        merge(data)
        self.assertEqual(data, [[1, 4], [2, 3]])

if __name__ == '__main__':
    unittest.main()
