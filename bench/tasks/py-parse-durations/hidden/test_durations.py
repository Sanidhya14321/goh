import unittest
from durations import parse_duration as p

class D(unittest.TestCase):
    def test_forms(self):
        self.assertEqual(p('1h30m'), 5400)
        self.assertEqual(p('45s'), 45)
        self.assertEqual(p('2h'), 7200)
        self.assertEqual(p('1d2h'), 93600)
        self.assertEqual(p('90'), 90)
    def test_bad(self):
        for bad in ('', 'abc', '5x', '1h-2m', 'h'):
            with self.assertRaises(ValueError, msg=bad):
                p(bad)

if __name__ == '__main__':
    unittest.main()
