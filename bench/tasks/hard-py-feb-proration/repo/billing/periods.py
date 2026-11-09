import calendar
from datetime import date


def days_in_month(d):
    return calendar.monthrange(d.year, d.month)[1]


def remaining_fraction(change_day):
    """Fraction of change_day's month remaining, counting change_day itself."""
    total = days_in_month(change_day)
    remaining = total - change_day.day + 1
    return remaining / total
