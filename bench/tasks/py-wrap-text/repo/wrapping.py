def wrap(text, width):
    lines, line = [], ''
    for word in text.split(' '):
        if len(line) + len(word) + 1 > width:
            lines.append(line)
            line = word
        else:
            line = (line + ' ' + word).strip()
    return lines
