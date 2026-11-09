import unittest
from billing.payments import charge_with_retry, TimeoutError_, PaymentError


class Gateway:
    def __init__(self, timeouts_after_success=0, fail_with=None):
        self.charges = []
        self.timeouts = timeouts_after_success
        self.fail_with = fail_with

    def charge(self, invoice, amount):
        if self.fail_with:
            raise self.fail_with
        self.charges.append(invoice)
        if self.timeouts:
            self.timeouts -= 1
            raise TimeoutError_()
        return "ch_%d" % len(self.charges)

    def lookup(self, invoice):
        return "ch_found" if invoice in self.charges else None


class Payments(unittest.TestCase):
    def test_timeout_after_success_is_not_recharged(self):
        g = Gateway(timeouts_after_success=1)
        self.assertEqual(charge_with_retry(g, "inv1", 10), "ch_found")
        self.assertEqual(g.charges, ["inv1"])

    def test_other_errors_not_retried(self):
        g = Gateway(fail_with=ValueError("card declined"))
        with self.assertRaises(ValueError):
            charge_with_retry(g, "inv2", 10)

    def test_plain_success(self):
        self.assertEqual(charge_with_retry(Gateway(), "inv3", 5), "ch_1")


if __name__ == "__main__":
    unittest.main()
