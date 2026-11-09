def paginate(items, page, size):
    start = page * size
    return items[start:start + size]

def page_count(items, size):
    return len(items) // size
