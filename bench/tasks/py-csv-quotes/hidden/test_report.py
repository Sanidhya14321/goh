import unittest
from report import load_rows

class Report(unittest.TestCase):
    def test_quoted(self):
        rows = load_rows('name,city\n"Smith, Jane",Paris\nBob,"Rome, IT"\n')
        self.assertEqual(rows, [{'name': 'Smith, Jane', 'city': 'Paris'}, {'name': 'Bob', 'city': 'Rome, IT'}])
    def test_escaped_quote(self):
        rows = load_rows('q\n"say ""hi"""\n')
        self.assertEqual(rows, [{'q': 'say "hi"'}])

if __name__ == '__main__':
    unittest.main()
