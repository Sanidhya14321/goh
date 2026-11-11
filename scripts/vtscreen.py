"""A small terminal screen emulator for the PTY test suites.

The TUI renderer redraws only the cells that changed, so text on screen can be
assembled from several writes separated by cursor moves (for example "Allow "
kept from an earlier prompt and "shell?" written later). Matching the raw byte
stream misses that; this keeps a character grid of what the terminal shows so
tests can match the screen instead. It implements the control sequences the
renderer emits (cursor movement, erasing, scroll regions) and ignores styling.
"""

import re

_CSI = re.compile(rb'\x1b\[([?<>=]?)([0-9;:]*)([ -/]*)([@-~])')
_OSC = re.compile(rb'\x1b\][^\x07\x1b]*(\x07|\x1b\\)')


class Screen:
    def __init__(self, rows=24, cols=80):
        self.rows, self.cols = rows, cols
        self.grid = [[' '] * cols for _ in range(rows)]
        self.y = self.x = 0
        self.top, self.bottom = 0, rows - 1
        self.saved = (0, 0)
        self.pending = b''

    def text(self):
        return '\n'.join(''.join(row).rstrip() for row in self.grid)

    def _scroll_up(self, n=1):
        for _ in range(n):
            del self.grid[self.top]
            self.grid.insert(self.bottom, [' '] * self.cols)

    def _scroll_down(self, n=1):
        for _ in range(n):
            del self.grid[self.bottom]
            self.grid.insert(self.top, [' '] * self.cols)

    def _clamp(self):
        self.y = min(max(self.y, 0), self.rows - 1)
        self.x = min(max(self.x, 0), self.cols - 1)

    def _linefeed(self):
        if self.y == self.bottom:
            self._scroll_up()
        else:
            self.y = min(self.y + 1, self.rows - 1)

    def _put(self, ch):
        if self.x >= self.cols:
            self.x = 0
            self._linefeed()
        self.grid[self.y][self.x] = ch
        self.x += 1

    def feed(self, data):
        data = self.pending + bytes(data)
        self.pending = b''
        i = 0
        while i < len(data):
            b = data[i]
            if b == 0x1b:
                m = _CSI.match(data, i)
                if m:
                    self._csi(m.group(1), m.group(2).decode(), m.group(4).decode())
                    i = m.end()
                    continue
                m = _OSC.match(data, i)
                if m:
                    i = m.end()
                    continue
                if i + 1 >= len(data) or (data[i + 1:i + 2] in (b'[', b']') and b'\x07' not in data[i:] and not re.search(rb'[@-~]', data[i + 2:])):
                    self.pending = data[i:]  # incomplete sequence: wait for more
                    return
                code = data[i + 1:i + 2]
                if code == b'7':
                    self.saved = (self.y, self.x)
                elif code == b'8':
                    self.y, self.x = self.saved
                elif code == b'M':
                    if self.y == self.top:
                        self._scroll_down()
                    else:
                        self.y = max(0, self.y - 1)
                elif code == b'D':
                    self._linefeed()
                elif code == b'E':
                    self.x = 0
                    self._linefeed()
                i += 2
                continue
            if b == 0x0d:
                self.x = 0
            elif b == 0x0a:
                self._linefeed()
            elif b == 0x08:
                self.x = max(0, self.x - 1)
            elif b == 0x09:
                self.x = min(self.cols - 1, (self.x // 8 + 1) * 8)
            elif b < 0x20 or b == 0x7f:
                pass
            else:
                # One UTF-8 character; a split character waits for more data.
                n = 1 if b < 0x80 else 2 if b >> 5 == 6 else 3 if b >> 4 == 14 else 4 if b >> 3 == 30 else 1
                if i + n > len(data):
                    self.pending = data[i:]
                    return
                self._put(data[i:i + n].decode('utf-8', 'replace'))
                i += n
                continue
            i += 1

    def _csi(self, private, params, final):
        if private:
            return  # modes (cursor visibility, alternate screen, mouse)
        args = [int(p) if p.isdigit() else 0 for p in params.replace(':', ';').split(';')] if params else []
        n = args[0] if args and args[0] else 1
        if final in 'Hf':
            self.y = (args[0] if args and args[0] else 1) - 1
            self.x = (args[1] if len(args) > 1 and args[1] else 1) - 1
        elif final == 'A':
            self.y -= n
        elif final in 'Be':
            self.y += n
        elif final in 'Ca':
            self.x += n
        elif final == 'D':
            self.x -= n
        elif final in 'G`':
            self.x = n - 1
        elif final == 'd':
            self.y = n - 1
        elif final == 'E':
            self.y, self.x = self.y + n, 0
        elif final == 'F':
            self.y, self.x = self.y - n, 0
        elif final == 'K':
            mode = args[0] if args else 0
            row = self.grid[self.y]
            lo, hi = {0: (self.x, self.cols), 1: (0, self.x + 1), 2: (0, self.cols)}.get(mode, (0, 0))
            row[lo:hi] = [' '] * (hi - lo)
        elif final == 'J':
            mode = args[0] if args else 0
            if mode in (2, 3):
                self.grid = [[' '] * self.cols for _ in range(self.rows)]
            elif mode == 0:
                self.grid[self.y][self.x:] = [' '] * (self.cols - self.x)
                for r in range(self.y + 1, self.rows):
                    self.grid[r] = [' '] * self.cols
            elif mode == 1:
                for r in range(self.y):
                    self.grid[r] = [' '] * self.cols
                self.grid[self.y][:self.x + 1] = [' '] * (self.x + 1)
        elif final == 'X':
            row = self.grid[self.y]
            end = min(self.cols, self.x + n)
            row[self.x:end] = [' '] * (end - self.x)
        elif final == 'P':
            row = self.grid[self.y]
            del row[self.x:self.x + n]
            row.extend([' '] * (self.cols - len(row)))
        elif final == '@':
            row = self.grid[self.y]
            row[self.x:self.x] = [' '] * n
            del row[self.cols:]
        elif final == 'r':
            self.top = (args[0] if args and args[0] else 1) - 1
            self.bottom = (args[1] if len(args) > 1 and args[1] else self.rows) - 1
            self.top, self.bottom = max(0, self.top), min(self.rows - 1, self.bottom)
            self.y = self.x = 0
        elif final == 'S':
            self._scroll_up(n)
        elif final == 'T':
            self._scroll_down(n)
        elif final == 'L':
            if self.top <= self.y <= self.bottom:
                for _ in range(n):
                    del self.grid[self.bottom]
                    self.grid.insert(self.y, [' '] * self.cols)
        elif final == 'M':
            if self.top <= self.y <= self.bottom:
                for _ in range(n):
                    del self.grid[self.y]
                    self.grid.insert(self.bottom, [' '] * self.cols)
        self._clamp()
