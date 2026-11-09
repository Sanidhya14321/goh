def load_rows(text):
    """Parse CSV text with a header row into a list of dicts."""
    lines = [l for l in text.splitlines() if l.strip()]
    header = lines[0].split(',')
    return [dict(zip(header, l.split(','))) for l in lines[1:]]
