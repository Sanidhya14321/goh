#!/usr/bin/env python3
"""First-run provider setup with a disposable home and authenticated loopback."""
import http.server
import json
import os
import pathlib
import pty
import select
import stat
import subprocess
import tempfile
import termios
import threading
import time

binary = pathlib.Path(os.environ.get("GOH_TEST_BINARY", pathlib.Path(__file__).resolve().parents[1] / "bin/goh"))
key = "goh-test-provider-key"

requests = []
prompts = []

class Provider(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        requests.append(self.path)
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        prompts.append(body["messages"][-1]["content"])
        if (self.path, self.headers.get("Authorization"), body["model"]) != (
            "/proxy/v1/chat/completions", "Bearer " + key, "setup-fixture"
        ):
            self.send_error(401)
            return
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()
        for data in (
            {"choices": [{"index": 0, "delta": {"content": "Setup authenticated"}, "finish_reason": None}]},
            {"choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}]},
        ):
            self.wfile.write(b"data: " + json.dumps(data).encode() + b"\n\n")
        self.wfile.write(b"data: [DONE]\n\n")

    def log_message(self, *_):
        pass

with tempfile.TemporaryDirectory(prefix="goh-setup-") as temporary, http.server.ThreadingHTTPServer(("127.0.0.1", 0), Provider) as server:
    threading.Thread(target=server.serve_forever, daemon=True).start()
    root = pathlib.Path(temporary)
    project = root / "project"
    project.mkdir()
    env = {k: v for k, v in os.environ.items() if not k.startswith("HARNESS_") and k != "OPENAI_API_KEY"}
    env.update(HOME=str(root), XDG_CONFIG_HOME=str(root / ".config"), TERM="xterm-256color")
    master, slave = pty.openpty()
    original = termios.tcgetattr(slave)
    app = subprocess.Popen([binary], cwd=project, env=env, stdin=slave, stdout=slave, stderr=slave)
    output = bytearray()

    def wait_for(text, start=0):
        deadline = time.monotonic() + 8
        while text not in output[start:] and time.monotonic() < deadline:
            if select.select([master], [], [], .1)[0]:
                output.extend(os.read(master, 65536))
        assert text in output[start:], (text, bytes(output[-900:]))

    try:
        wait_for(b"GoHarness")
        wait_for(b"Chat setup required")
        assert not requests
        os.write(master, b"Should stay blocked\r")
        time.sleep(.1)
        assert not requests
        os.write(master, b"\x10\r")
        wait_for(b"Provider setup")
        os.write(master, f"http://127.0.0.1:{server.server_port}/proxy/v1".encode() + b"\t\tsetup-fixture\t\t\t" + b"incorrect-test-key")
        wait_for("••••".encode())
        os.write(master, b"\x13")
        wait_for(b"Agent: build")
        wait_for(b"Ctrl+P commands")
        time.sleep(.1)
        os.write(master, b"Helo from setup\x01\x1b[C\x1b[Cl\x05\r")
        wait_for(b"Provider authentication failed")
        assert prompts == ["Hello from setup"], prompts
        journal_before, = root.rglob("*.jsonl")
        count = len(requests)
        os.write(master, b"Should stay blocked\r")
        time.sleep(.1)
        assert len(requests) == count
        pos = len(output)
        os.write(master, b"\x10\r")
        wait_for(b"Provider setup", pos)
        os.write(master, b"\t\t\t\t\t" + key.encode() + b"\x13")
        wait_for(b"Agent: build", pos)
        os.write(master, b"Retry after fixing key\r")
        wait_for(b"Setup authenticated")
        assert list(root.rglob("*.jsonl")) == [journal_before]
        wait_for(b"completed")
        count = len(requests)
        os.write(master, b"\x1b[A\r")
        deadline = time.monotonic()+8
        while len(requests) == count and time.monotonic() < deadline:
            time.sleep(.01)
        assert len(requests) == count+1 and prompts[-1] == "Retry after fixing key", prompts
        # Wait for the recalled prompt's run to settle before navigating.
        time.sleep(.1)
        pos = len(output)
        os.write(master, b"/settings\r")
        wait_for(b"Edit advanced settings", pos)
        os.write(master, b"s")
        wait_for(b"Advanced configuration", pos)
        pos = len(output)
        os.write(master, b"\x1b")
        wait_for(b"Run a GoHarness command", pos)
        pos = len(output)
        os.write(master, b"q")
        wait_for(b"Agent: build", pos)
        os.write(master, b"\x04")
        assert app.wait(timeout=8) == 0
        restored = termios.tcgetattr(slave)
        # Darwin can set PENDIN when returning to canonical mode.
        pending = getattr(termios, "PENDIN", 0)
        restored[3] &= ~pending
        original[3] &= ~pending
        assert restored == original
        assert key.encode() not in output
        config, = root.rglob("config.json")
        assert json.loads(config.read_text())["model"] == "setup-fixture"
        assert key not in config.read_text()
        secret, = root.rglob("*.key")
        assert stat.S_IMODE(secret.stat().st_mode) == 0o600 and secret.read_text() == key
        run = subprocess.run([binary, "run", "--prompt", "Check persisted key"], cwd=project, env=env, capture_output=True, text=True, timeout=8)
        assert run.returncode == 0 and "Setup authenticated" in run.stdout, run.stderr
        blocked = subprocess.run([binary, "run", "--prompt", "Check workspace boundary"], cwd=root, env=env, capture_output=True, text=True, timeout=8)
        assert blocked.returncode != 0 and "credential storage scope" in blocked.stderr
        print(json.dumps({"setup": "passed", "chat_opens_before_setup": True, "unconfigured_chat_makes_no_requests": True, "key_masked": True, "cursor_editing_verified": True, "history_submission_verified": True, "authentication_error_recoverable": True, "session_preserved": True, "saved_key_used_by_cli": True, "settings_reachable_from_chat": True, "terminal_restored": True}))
    finally:
        if app.poll() is None:
            app.kill()
            app.wait()
        os.close(master)
        os.close(slave)
        server.shutdown()
