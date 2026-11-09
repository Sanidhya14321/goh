class TaskList:
    def __init__(self, items=[]):
        self.items = items

    def add(self, text):
        self.items.append(text)
        return self

    def __len__(self):
        return len(self.items)
