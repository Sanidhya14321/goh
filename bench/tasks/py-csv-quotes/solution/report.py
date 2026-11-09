import csv
import io

def load_rows(text):
    """Parse CSV text with a header row into a list of dicts."""
    return [dict(r) for r in csv.DictReader(io.StringIO(text))]
