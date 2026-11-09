import unittest
from todo import TaskList

class T(unittest.TestCase):
    def test_add(self):
        self.assertEqual(len(TaskList().add('a')), 1)

if __name__ == '__main__':
    unittest.main()
