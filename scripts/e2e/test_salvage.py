#!/usr/bin/env python3
"""`goh chat --session ID` on a damaged session: the recovery question is
asked on the normal terminal (the chat's screen is handed back for it), and
answering yes opens the recovered conversation in the chat."""
import fcntl, http.server, json, os, pathlib, pty, select, struct, subprocess, sys, tempfile, termios, threading, time

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from vtscreen import Screen

binary = pathlib.Path(os.environ.get("GOH_TEST_BINARY", pathlib.Path(__file__).resolve().parents[2] / "bin/goh"))
key = "salvage-key"


class Provider(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        self.rfile.read(int(self.headers["Content-Length"]))
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()
        for data in ({"choices": [{"index": 0, "delta": {"content": "Original reply"}}]}, {"choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}]}):
            self.wfile.write(b"data: " + json.dumps(data).encode() + b"\n\n")
        self.wfile.write(b"data: [DONE]\n\n")

    def log_message(self, *_):
        pass


with tempfile.TemporaryDirectory(prefix="goh-salvage-") as temporary, http.server.ThreadingHTTPServer(("127.0.0.1", 0), Provider) as server:
    threading.Thread(target=server.serve_forever, daemon=True).start()
    root = pathlib.Path(temporary)
    project = root / "project"
    project.mkdir()
    env = {k: v for k, v in os.environ.items() if not k.startswith("HARNESS_") and k != "OPENAI_API_KEY"}
    env.update(HOME=str(root), XDG_CONFIG_HOME=str(root / ".config"), TERM="xterm-256color", SALVAGE_KEY=key)
    config_dir = root / ("Library/Application Support" if sys.platform == "darwin" else ".config") / "independent-harness"
    config_dir.mkdir(parents=True)
    url = f"http://127.0.0.1:{server.server_port}/v1"
    (config_dir / "config.json").write_text(json.dumps({"base_url": url, "model": "fixture", "context_window": 131072, "api_key_env": "SALVAGE_KEY"}))
    for prompt in ("First turn", "Second turn"):
        args = ["run", "--prompt", prompt, "--format", "json"]
        journals = list(root.rglob("*.jsonl"))
        if journals:
            args = ["resume", journals[0].stem, "--prompt", prompt, "--format", "json"]
        subprocess.run([str(binary), *args], cwd=project, env=env, check=True, capture_output=True, timeout=30)
    journal, = root.rglob("*.jsonl")
    # Damage a record in the middle; the end of the file stays intact.
    lines = journal.read_bytes().splitlines(keepends=True)
    assert len(lines) > 4, len(lines)
    # The first turn stays readable; a record of the second turn is broken.
    second = next(i for i, line in enumerate(lines) if b"Second turn" in line)
    lines[second + 1] = b'{"not a record\n'
    damaged = b"".join(lines)
    journal.write_bytes(damaged)

    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 24, 100, 0, 0))
    screen = Screen(24, 100)
    raw = bytearray()
    app = subprocess.Popen([str(binary), "chat", "--session", journal.stem], cwd=project, env=env, stdin=slave, stdout=slave, stderr=slave)
    os.close(slave)

    def pump(seconds):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            if select.select([master], [], [], .05)[0]:
                try:
                    chunk = os.read(master, 65536)
                except OSError:
                    return
                raw.extend(chunk)
                screen.feed(chunk)

    def wait_for(needle, timeout=10):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if needle in screen.text() or needle.encode() in raw:
                return
            pump(.05)
        raise AssertionError(f"missing {needle!r} on screen:\n{screen.text()}")

    try:
        wait_for("is damaged")
        os.write(master, b"y\r")
        wait_for("Recovered")
        # The chat comes back with the readable conversation.
        wait_for("Ctrl+D")
        wait_for("First turn")
        assert journal.read_bytes() == damaged, "the damaged journal was changed"
        recovered = [p for p in root.rglob("*.jsonl") if p != journal]
        assert len(recovered) == 1, recovered
        os.write(master, b"\x04")
        app.wait(timeout=10)
        pump(.3)
        assert app.returncode == 0, app.returncode
        assert b"\x1b[?1049l" in raw, "terminal not restored"
    finally:
        if app.poll() is None:
            app.kill()
            app.wait()
        os.close(master)
print(json.dumps({"salvage": "passed", "question_on_normal_screen": True, "recovered_session_opened": True, "damaged_file_unchanged": True}))
