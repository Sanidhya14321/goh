import pathlib, unittest
from shop import pricing
from shop.invoice import invoice_line

class Rename(unittest.TestCase):
    def test_new_name(self):
        self.assertEqual(pricing.compute_total([1, 2]), 3)
    def test_alias(self):
        self.assertEqual(pricing.calc([1, 2]), 3)
    def test_callers_updated(self):
        for f in ('checkout.py', 'invoice.py'):
            src = pathlib.Path('shop', f).read_text()
            self.assertIn('compute_total', src, f)
            self.assertNotIn('calc(', src, f)
        self.assertEqual(invoice_line([1, 2]), 'Total: 3.00')

if __name__ == '__main__':
    unittest.main()
