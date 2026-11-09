import unittest
from decimal import Decimal
from cart import Cart

class CartHidden(unittest.TestCase):
    def test_half_up(self):
        c = Cart(); c.add(0.10, 1); c.add(0.05, 1)
        self.assertEqual(Decimal(str(c.total(50))), Decimal('0.08'))
    def test_full_discount(self):
        c = Cart(); c.add(19.99, 3)
        self.assertEqual(c.total(100), 0)
        self.assertGreaterEqual(c.total(120), 0)
    def test_exact(self):
        c = Cart(); c.add(1.10, 3)
        self.assertEqual(Decimal(str(c.total(10))), Decimal('2.97'))

if __name__ == '__main__':
    unittest.main()
