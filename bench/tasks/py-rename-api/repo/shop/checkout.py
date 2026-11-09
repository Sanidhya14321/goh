from shop.pricing import calc

def checkout(prices):
    return {'total': calc(prices, tax=0.2)}
