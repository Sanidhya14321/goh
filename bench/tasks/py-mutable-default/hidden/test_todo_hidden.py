import unittest
from todo import TaskList

class Hidden(unittest.TestCase):
    def test_independent(self):
        a = TaskList(); b = TaskList()
        a.add('x')
        self.assertEqual(len(b), 0)
    def test_given_list_copied(self):
        src = ['p']
        t = TaskList(src); t.add('q')
        self.assertEqual(src, ['p'])

if __name__ == '__main__':
    unittest.main()
