import unittest
from cart import Cart

class Cart0(unittest.TestCase):
    def test_plain(self):
        c = Cart(); c.add(2.5, 2)
        self.assertEqual(c.total(), 5.0)

if __name__ == '__main__':
    unittest.main()
