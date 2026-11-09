def paginate(items, page, size):
    if page < 1:
        return []
    start = (page - 1) * size
    return items[start:start + size]

def page_count(items, size):
    return (len(items) + size - 1) // size
