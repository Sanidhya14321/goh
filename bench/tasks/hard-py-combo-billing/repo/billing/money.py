from decimal import Decimal, ROUND_HALF_UP

CENT = Decimal("0.01")


def to_money(value):
    """Return value as a Decimal rounded half up to the cent."""
    return Decimal(str(value)).quantize(CENT, rounding=ROUND_HALF_UP)
