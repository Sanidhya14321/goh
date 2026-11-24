#!/usr/bin/env python3
"""Interactive latency, measured from outside: submit to first visible text,
and final text to idle. A loopback provider holds its first text for 400 ms
and the rest for 300 ms, so the first interval must reflect that delay and
the last must be the harness's own (short) finishing work."""
import fcntl, http.server, json, os, pathlib, pty, select, struct, subprocess, sys, tempfile, termios, threading, time

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from vtscreen import Screen

binary = pathlib.Path(os.environ.get("GOH_TEST_BINARY", pathlib.Path(__file__).resolve().parents[2] / "bin/goh"))
FIRST_DELAY, REST_DELAY = 0.4, 0.3


class Provider(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"data":[{"id":"fixture"}]}')

    def do_POST(self):
        self.rfile.read(int(self.headers["Content-Length"]))
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()

        def send(data):
            self.wfile.write(b"data: " + json.dumps(data).encode() + b"\n\n")
            self.wfile.flush()

        time.sleep(FIRST_DELAY)
        send({"choices": [{"index": 0, "delta": {"content": "Opening words"}}]})
        time.sleep(REST_DELAY)
        send({"choices": [{"index": 0, "delta": {"content": " and closing words."}}]})
        send({"choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}})
        self.wfile.write(b"data: [DONE]\n\n")

    def log_message(self, *_):
        pass


with tempfile.TemporaryDirectory(prefix="goh-latency-") as temporary, http.server.ThreadingHTTPServer(("127.0.0.1", 0), Provider) as server:
    threading.Thread(target=server.serve_forever, daemon=True).start()
    root = pathlib.Path(temporary)
    project = root / "project"
    project.mkdir()
    env = {k: v for k, v in os.environ.items() if not k.startswith(("HARNESS_", "GOH_")) and k != "OPENAI_API_KEY"}
    env.update(HOME=str(root), XDG_CONFIG_HOME=str(root / ".config"), TERM="xterm-256color",
               HARNESS_BASE_URL=f"http://127.0.0.1:{server.server_port}/v1", HARNESS_MODEL="fixture",
               HARNESS_CONTEXT_WINDOW="131072", HARNESS_API_KEY_ENV="UNSET_FIXTURE")
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 30, 100, 0, 0))
    screen = Screen(30, 100)
    app = subprocess.Popen([str(binary), "chat", "--project", str(project)], cwd=project, env=env, stdin=slave, stdout=slave, stderr=slave)
    os.close(slave)

    def wait_for(needle, timeout=10):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if needle in screen.text():
                return time.monotonic()
            if select.select([master], [], [], .005)[0]:
                try:
                    screen.feed(os.read(master, 65536))
                except OSError:
                    break
        raise AssertionError(f"missing {needle!r} on screen:\n{screen.text()}")

    try:
        wait_for("Ask anything")
        os.write(master, b"say something")
        wait_for("say something")
        submitted = time.monotonic()
        os.write(master, b"\r")
        first = wait_for("Opening words")
        final = wait_for("closing words.")
        idle = wait_for("completed")
        result = {"submit_to_first_text_ms": round((first - submitted) * 1000), "first_to_final_text_ms": round((final - first) * 1000),
                  "final_text_to_idle_ms": round((idle - final) * 1000)}
        # The provider's delay is where the first interval goes; what is
        # left over is the harness (request, stream, render).
        assert FIRST_DELAY * 1000 <= result["submit_to_first_text_ms"] < FIRST_DELAY * 1000 + 1000, result
        assert result["first_to_final_text_ms"] >= REST_DELAY * 1000 - 50, result
        assert result["final_text_to_idle_ms"] < 1000, result
        print(json.dumps({"latency": "passed", **result}))
    finally:
        app.kill()
        app.wait()
