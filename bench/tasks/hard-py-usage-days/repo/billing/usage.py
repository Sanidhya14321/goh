from collections import defaultdict
from datetime import datetime, timezone


def daily_totals(lines):
    """Sum usage per UTC day from lines like
    '2026-03-01T23:30:00-05:00 api 12' (ISO time with offset, meter, units).
    Days are UTC calendar days."""
    totals = defaultdict(int)
    for line in lines:
        line = line.strip()
        if not line:
            continue
        stamp, meter, units = line.split()
        totals[(stamp[:10], meter)] += int(units)
    return dict(totals)
