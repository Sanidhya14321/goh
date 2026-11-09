from decimal import Decimal, ROUND_HALF_UP

def apply_discount(amount, percent):
    """Return amount after a percentage discount, exact to the cent, never negative."""
    value = Decimal(str(amount)) * (Decimal(100) - Decimal(str(percent))) / Decimal(100)
    value = max(value, Decimal(0))
    return value.quantize(Decimal('0.01'), rounding=ROUND_HALF_UP)
