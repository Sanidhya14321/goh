def compute_total(prices, tax=0.0):
    return round(sum(prices) * (1 + tax), 2)


def calc(prices, tax=0.0):
    """Deprecated: use compute_total."""
    return compute_total(prices, tax)
