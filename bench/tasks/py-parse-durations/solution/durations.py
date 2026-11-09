import re

_UNITS = {'d': 86400, 'h': 3600, 'm': 60, 's': 1}

def parse_duration(text):
    if re.fullmatch(r'\d+', text or ''):
        return int(text)
    if not text or not re.fullmatch(r'(\d+[dhms])+', text):
        raise ValueError('invalid duration: %r' % text)
    return sum(int(n) * _UNITS[u] for n, u in re.findall(r'(\d+)([dhms])', text))
