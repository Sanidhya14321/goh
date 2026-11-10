import unittest
from datetime import date
from decimal import Decimal
from billing.rates import RateCache, convert


class Rates(unittest.TestCase):
    def test_rate_per_day(self):
        calls = []
        table = {("EUR", date(2026, 3, 1)): Decimal("1.10"), ("EUR", date(2026, 3, 2)): Decimal("1.20")}

        def fetch(cur, day):
            calls.append((cur, day))
            return table[(cur, day)]
        cache = RateCache(fetch)
        self.assertEqual(convert(Decimal("100"), "EUR", date(2026, 3, 1), cache), Decimal("110.00"))
        self.assertEqual(convert(Decimal("100"), "EUR", date(2026, 3, 2), cache), Decimal("120.00"))
        self.assertEqual(convert(Decimal("50"), "EUR", date(2026, 3, 1), cache), Decimal("55.00"))
        self.assertEqual(len(calls), 2, "rates must still be cached per day")

    def test_usd_untouched(self):
        self.assertEqual(convert(Decimal("3.333"), "USD", date(2026, 3, 1), RateCache(None)), Decimal("3.33"))


if __name__ == "__main__":
    unittest.main()
