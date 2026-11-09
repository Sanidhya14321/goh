import re
import unicodedata

def slugify(text):
    """Turn text into a URL slug."""
    text = unicodedata.normalize('NFKD', text).encode('ascii', 'ignore').decode('ascii').lower()
    return re.sub(r'[^a-z0-9]+', '-', text).strip('-')
