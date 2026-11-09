def median(values):
    """Return the median of a list of numbers."""
    if not values:
        raise ValueError('median of empty list')
    ordered = sorted(values)
    mid = len(ordered) // 2
    if len(ordered) % 2:
        return ordered[mid]
    return (ordered[mid - 1] + ordered[mid]) / 2
