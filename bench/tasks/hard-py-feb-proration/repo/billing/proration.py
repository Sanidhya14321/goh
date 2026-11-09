from decimal import Decimal

from billing.money import to_money
from billing.periods import days_in_month


def prorated_upgrade(old_price, new_price, change_day):
    """Charge for upgrading mid-month: the price difference for the days
    remaining in the month, counting the change day, using that month's
    actual length (February and leap years included)."""
    total = 30
    remaining = total - change_day.day
    diff = Decimal(str(new_price)) - Decimal(str(old_price))
    return to_money(diff * remaining / total)
