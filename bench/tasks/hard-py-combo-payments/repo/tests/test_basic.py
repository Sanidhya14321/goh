import unittest
from decimal import Decimal
from billing.money import to_money


class Money(unittest.TestCase):
    def test_round(self):
        self.assertEqual(to_money(2.675), Decimal("2.68"))


if __name__ == "__main__":
    unittest.main()
