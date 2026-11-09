from shop.pricing import compute_total

def checkout(prices):
    return {'total': compute_total(prices, tax=0.2)}
