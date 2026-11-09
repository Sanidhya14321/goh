class TaskList:
    def __init__(self, items=None):
        self.items = list(items) if items is not None else []

    def add(self, text):
        self.items.append(text)
        return self

    def __len__(self):
        return len(self.items)
