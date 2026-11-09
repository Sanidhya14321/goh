import unittest
from datetime import date
from decimal import Decimal
from billing.proration import prorated_upgrade


class Proration(unittest.TestCase):
    def test_leap_february(self):
        self.assertEqual(prorated_upgrade(10, 39, date(2028, 2, 15)), Decimal("15.00"))

    def test_last_day(self):
        self.assertEqual(prorated_upgrade(0, 31, date(2026, 1, 31)), Decimal("1.00"))

    def test_first_day_full_month(self):
        self.assertEqual(prorated_upgrade(5, 25, date(2026, 4, 1)), Decimal("20.00"))

    def test_rounding(self):
        self.assertEqual(prorated_upgrade(0, 10, date(2026, 3, 10)), Decimal("7.10"))


if __name__ == "__main__":
    unittest.main()
