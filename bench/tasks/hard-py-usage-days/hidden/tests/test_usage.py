import unittest
from billing.usage import daily_totals


class Usage(unittest.TestCase):
    def test_utc_days(self):
        lines = [
            "2026-03-01T23:30:00-05:00 api 12",
            "2026-03-02T01:00:00+00:00 api 3",
            "2026-03-02T09:00:00+09:00 api 5",
            "",
            "2026-03-01T10:00:00+00:00 storage 7",
        ]
        self.assertEqual(daily_totals(lines), {
            ("2026-03-02", "api"): 20,
            ("2026-03-01", "storage"): 7,
        })


if __name__ == "__main__":
    unittest.main()
