import unittest
from shop.checkout import checkout

class Shop(unittest.TestCase):
    def test_checkout(self):
        self.assertEqual(checkout([10, 5])['total'], 18.0)

if __name__ == '__main__':
    unittest.main()
