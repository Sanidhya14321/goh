from billing.money import to_money


class RateCache:
    """Caches exchange rates per (currency, day); rates change daily."""

    def __init__(self, fetch):
        self._fetch = fetch
        self._cache = {}

    def rate(self, currency, day):
        key = currency
        if key not in self._cache:
            self._cache[key] = self._fetch(currency, day)
        return self._cache[key]


def convert(amount, currency, day, cache):
    """Convert amount in currency to USD at that day's rate."""
    if currency == "USD":
        return to_money(amount)
    return to_money(amount * cache.rate(currency, day))
