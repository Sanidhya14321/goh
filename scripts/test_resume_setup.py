#!/usr/bin/env python3
"""`goh chat --session ID` before a provider is configured: completing setup
from the blocked screen must open that session, not a new one."""
import fcntl, http.server, json, os, pathlib, pty, select, struct, subprocess, sys, tempfile, termios, threading, time

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from vtscreen import Screen

binary = pathlib.Path(os.environ.get("GOH_TEST_BINARY", pathlib.Path(__file__).resolve().parents[1] / "bin/goh"))
key = "resume-setup-key"
prompts = []


class Provider(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        if self.headers.get("Authorization") != "Bearer " + key:
            self.send_error(401)
            return
        prompts.append(body["messages"][-1]["content"])
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()
        for data in ({"choices": [{"index": 0, "delta": {"content": "Resumed reply"}}]}, {"choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}]}):
            self.wfile.write(b"data: " + json.dumps(data).encode() + b"\n\n")
        self.wfile.write(b"data: [DONE]\n\n")

    def log_message(self, *_):
        pass


with tempfile.TemporaryDirectory(prefix="goh-resume-setup-") as temporary, http.server.ThreadingHTTPServer(("127.0.0.1", 0), Provider) as server:
    threading.Thread(target=server.serve_forever, daemon=True).start()
    root = pathlib.Path(temporary)
    project = root / "project"
    project.mkdir()
    env = {k: v for k, v in os.environ.items() if not k.startswith("HARNESS_") and k != "OPENAI_API_KEY"}
    env.update(HOME=str(root), XDG_CONFIG_HOME=str(root / ".config"), TERM="xterm-256color")
    config_dir = root / ("Library/Application Support" if sys.platform == "darwin" else ".config") / "independent-harness"
    config_dir.mkdir(parents=True)
    url = f"http://127.0.0.1:{server.server_port}/v1"
    # A first session made with a working provider...
    (config_dir / "config.json").write_text(json.dumps({"base_url": url, "model": "fixture", "context_window": 131072, "api_key_env": "RESUME_SETUP_KEY"}))
    subprocess.run([str(binary), "run", "--prompt", "First turn", "--format", "json"], cwd=project, env=dict(env, RESUME_SETUP_KEY=key), check=True, capture_output=True, timeout=30)
    journal, = root.rglob("*.jsonl")
    # ...then no provider, and chat asked to resume it.
    (config_dir / "config.json").write_text("{}")
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 24, 80, 0, 0))
    screen = Screen(24, 80)
    app = subprocess.Popen([str(binary), "chat", "--session", journal.stem], cwd=project, env=env, stdin=slave, stdout=slave, stderr=slave)
    os.close(slave)

    def wait_for(needle, timeout=10):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if needle in screen.text():
                return
            if select.select([master], [], [], .05)[0]:
                try:
                    screen.feed(os.read(master, 65536))
                except OSError:
                    break
        raise AssertionError(f"missing {needle!r} on screen:\n{screen.text()}")

    try:
        wait_for("Chat setup required")
        os.write(master, b"\x07")
        wait_for("Status")
        os.write(master, b"p")
        wait_for("Provider setup")
        os.write(master, url.encode() + b"\t\tfixture\t\t\t" + key.encode())
        time.sleep(.2)
        os.write(master, b"\x13")
        wait_for("Agent: build")
        os.write(master, b"Second turn")
        time.sleep(.2)
        os.write(master, b"\r")
        wait_for("Resumed reply")
        time.sleep(.3)
        journals = list(root.rglob("*.jsonl"))
        assert journals == [journal], f"setup opened another session: {journals}"
        assert prompts[-1] == "Second turn", prompts
        contents = journal.read_text()
        assert "First turn" in contents and "Second turn" in contents, "resumed session lacks the new turn"
        os.write(master, b"\x04")
        app.wait(timeout=10)
        print(json.dumps({"resume_setup": "passed", "requested_session_opened_after_setup": True}))
    finally:
        if app.poll() is None:
            app.kill()
