from shop import pricing

def invoice_line(prices):
    return 'Total: %.2f' % pricing.compute_total(prices)
