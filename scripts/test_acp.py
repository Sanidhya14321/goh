"""Exercise the compiled ACP agent over real stdio with a local provider."""

import http.server
import json
import os
from pathlib import Path
import select
import subprocess
import sys
import tempfile
import threading
import time


class Provider(http.server.BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_POST(self):
        request = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        history = request["messages"]
        latest = history[-1]
        delta, finish = {"content": "fixture done"}, "stop"
        if latest["role"] == "user" and latest["content"] in {"approve", "cancel"}:
            call = {
                "index": 0, "id": "reused-tool-id", "type": "function",
                "function": {
                    "name": "create_file",
                    "arguments": json.dumps({"path": latest["content"] + ".txt", "content": "fixture"}),
                },
            }
            delta, finish = {"tool_calls": [call]}, "tool_calls"
        if latest["content"] == "check history":
            assert any(m["role"] == "user" and m["content"] == "approve" for m in history)
            delta = {"content": "durable history restored"}
        wire = "".join("data: " + json.dumps({"choices": [choice]}) + "\n\n" for choice in [
            {"index": 0, "delta": delta},
            {"index": 0, "delta": {}, "finish_reason": finish},
        ]) + "data: [DONE]\n\n"
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Content-Length", str(len(wire)))
        self.end_headers()
        self.wfile.write(wire.encode())


class Client:
    def __init__(self, binary, project, config, sessions):
        env = {key: value for key, value in os.environ.items() if not key.startswith("HARNESS_")}
        self.proc = subprocess.Popen(
            [str(binary), "acp", "--project", str(project), "--config", str(config),
             "--session-dir", str(sessions)],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env,
        )
        self.buffer = b""

    def send(self, **frame):
        frame["jsonrpc"] = "2.0"
        self.proc.stdin.write(json.dumps(frame).encode() + b"\n")
        self.proc.stdin.flush()

    def read(self, deadline):
        while b"\n" not in self.buffer:
            remaining = max(0, deadline - time.monotonic())
            assert select.select([self.proc.stdout], [], [], remaining)[0], "ACP read timed out"
            chunk = os.read(self.proc.stdout.fileno(), 65536)
            assert chunk, "unexpected ACP EOF"
            self.buffer += chunk
        line, self.buffer = self.buffer.split(b"\n", 1)
        frame = json.loads(line)
        assert frame["jsonrpc"] == "2.0"
        return frame

    def until(self, predicate):
        deadline = time.monotonic() + 12
        frames = []
        while time.monotonic() < deadline:
            frame = self.read(deadline)
            if predicate(frame):
                return frame, frames
            frames.append(frame)
        raise AssertionError("expected ACP message was absent")

    def response(self, request_id):
        return self.until(lambda frame: frame.get("id") == request_id)

    def initialize(self):
        self.send(id=1, method="initialize", params={"protocolVersion": 7})
        frame, _ = self.response(1)
        assert frame["result"]["protocolVersion"] == 1

    def prompt(self, request_id, session_id, text):
        self.send(id=request_id, method="session/prompt", params={
            "sessionId": session_id, "prompt": [{"type": "text", "text": text}],
        })

    def shutdown(self, signal=False):
        if self.proc.poll() is None:
            if signal:
                self.proc.terminate()
            else:
                self.proc.stdin.close()
        try:
            self.proc.wait(timeout=12)
        except subprocess.TimeoutExpired:
            self.proc.kill()
            self.proc.wait()
            raise
        assert self.proc.returncode == 0, self.proc.stderr.read().decode(errors="replace")


def main(binary):
    provider = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Provider)
    threading.Thread(target=provider.serve_forever, daemon=True).start()
    try:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            project = root / "project"
            project.mkdir()
            sessions = root / "sessions"
            config = root / "config.json"
            config.write_text(json.dumps({
                "endpoint": f"http://127.0.0.1:{provider.server_port}/v1/chat/completions",
                "model": "fixture", "context_window": 131072, "api_key_env": "ACP_SMOKE_UNUSED",
            }))
            client = Client(binary, project, config, sessions)
            try:
                client.initialize()
                client.send(id=2, method="session/new", params={"cwd": str(root), "mcpServers": []})
                frame, _ = client.response(2)
                assert frame["error"]["code"] == -32602
                client.send(id=3, method="session/new", params={"cwd": str(project), "mcpServers": []})
                frame, _ = client.response(3)
                session_id = frame["result"]["sessionId"]
                client.prompt(4, session_id, "approve")
                permission, frames = client.until(lambda f: f.get("method") == "session/request_permission")
                call_id = permission["params"]["toolCall"]["toolCallId"]
                assert any(f.get("params", {}).get("update", {}).get("toolCallId") == call_id for f in frames)
                assert not (project / "approve.txt").exists()
                client.prompt(5, session_id, "overlap")
                frame, _ = client.response(5)
                assert frame["error"]["code"] == -32000
                client.send(id=permission["id"], result={"outcome": {"outcome": "selected", "optionId": "allow-once"}})
                frame, frames = client.response(4)
                assert frame["result"]["stopReason"] == "end_turn"
                assert (project / "approve.txt").read_text() == "fixture"
                assert any(f.get("params", {}).get("update", {}).get("status") == "completed" for f in frames)

                client.prompt(6, session_id, "cancel")
                permission, _ = client.until(lambda f: f.get("method") == "session/request_permission")
                assert permission["params"]["toolCall"]["toolCallId"] != call_id
                client.send(method="session/cancel", params={"sessionId": session_id})
                client.send(id=permission["id"], result={"outcome": {"outcome": "selected", "optionId": "allow-once"}})
                frame, _ = client.response(6)
                assert frame["result"]["stopReason"] == "cancelled"
                assert not (project / "cancel.txt").exists()
                client.send(id=7, method="session/close", params={"sessionId": session_id})
                frame, _ = client.response(7)
                assert frame["result"] == {}
            finally:
                client.shutdown()

            client = Client(binary, project, config, sessions)
            try:
                client.initialize()
                client.send(id=2, method="session/resume", params={
                    "sessionId": session_id, "cwd": str(project), "mcpServers": [],
                })
                frame, _ = client.response(2)
                assert frame["result"] == {}
                client.prompt(3, session_id, "check history")
                frame, frames = client.response(3)
                assert frame["result"]["stopReason"] == "end_turn"
                assert any(f.get("params", {}).get("update", {}).get("content", {}).get("text")
                           == "durable history restored" for f in frames)
                marker = project / "unexpected-mcp-start"
                client.send(id=4, method="session/new", params={"cwd": str(project), "mcpServers": [{
                    "name": "fixture", "command": "/bin/sh",
                    "args": ["-c", 'printf unexpected > "$ACP_MARKER"'],
                    "env": [{"name": "ACP_MARKER", "value": str(marker)}],
                }]})
                frame, _ = client.response(4)
                mcp_session = frame["result"]["sessionId"]
                assert not marker.exists()
                client.prompt(5, mcp_session, "mcp startup")
                permission, _ = client.until(lambda f: f.get("method") == "session/request_permission")
                assert "mcp_start:fixture" in permission["params"]["toolCall"]["title"]
                client.send(id=permission["id"], result={"outcome": {"outcome": "selected", "optionId": "reject-once"}})
                frame, _ = client.response(5)
                assert frame["result"]["stopReason"] == "refusal"
                assert not marker.exists()
            finally:
                client.shutdown(signal=True)
    finally:
        provider.shutdown()
        provider.server_close()
    print(json.dumps({"protocol": 1, "approval": "passed", "cancel_late_approval": "passed",
                      "project_scope": "passed", "durable_resume": "passed",
                      "client_mcp_startup_denial": "passed",
                      "eof_and_signal_shutdown": "passed"}, indent=2))


if __name__ == "__main__":
    main(Path(sys.argv[1] if len(sys.argv) > 1 else "bin/goh").resolve())
