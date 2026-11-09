def median(values):
    """Return the median of a list of numbers."""
    ordered = sorted(values)
    return ordered[len(ordered) // 2]
