from pricing import apply_discount

class Cart:
    def __init__(self):
        self.items = []

    def add(self, price, qty=1):
        self.items.append((price, qty))

    def total(self, discount_percent=0):
        subtotal = sum(p * q for p, q in self.items)
        return round(apply_discount(subtotal, discount_percent), 2)
