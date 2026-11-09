def wrap(text, width):
    words = []
    for w in text.split():
        while len(w) > width:
            words.append(w[:width])
            w = w[width:]
        words.append(w)
    lines, line = [], ''
    for w in words:
        if not line:
            line = w
        elif len(line) + 1 + len(w) <= width:
            line += ' ' + w
        else:
            lines.append(line)
            line = w
    if line:
        lines.append(line)
    return lines
