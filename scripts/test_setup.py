#!/usr/bin/env python3
"""First-run provider setup with a disposable home and authenticated loopback."""
import http.server
import json
import os
import pathlib
import pty
import select
import stat
import sys
import subprocess
import tempfile
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from vtscreen import Screen
import termios
import threading
import time
import fcntl
import struct

binary = pathlib.Path(os.environ.get("GOH_TEST_BINARY", pathlib.Path(__file__).resolve().parents[1] / "bin/goh"))
key = "goh-test-provider-key"

requests = []
prompts = []
models = []
toolsets = []

class Provider(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        requests.append(self.path)
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        prompts.append(body["messages"][-1]["content"])
        models.append(body["model"])
        toolsets.append({t["function"]["name"] for t in body.get("tools", [])})
        if (self.path != "/proxy/v1/chat/completions" or
                self.headers.get("Authorization") != "Bearer " + key or
                body["model"] not in ("setup-fixture", "fixture-fast")):
            self.send_error(401)
            return
        if prompts[-1] == "Compatibility fallback":
            if "stream_options" in body:
                self.send_response(400)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(b'{"error":{"code":"unsupported_parameter","param":"stream_options"}}')
                return
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"choices":[{"index":0,"message":{"role":"assistant","content":"Compatible JSON received"},"finish_reason":"stop"}],"usage":{"total_tokens":9}}')
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
    (project / "docs").mkdir()
    (project / "docs/query fixture.md").write_text("FILE_BODY_MUST_STAY_LOCAL")
    env = {k: v for k, v in os.environ.items() if not k.startswith("HARNESS_") and k not in ("OPENAI_API_KEY", "ANTHROPIC_API_KEY")}
    env["GOH_NO_BROWSER"] = "1"
    env.update(HOME=str(root), XDG_CONFIG_HOME=str(root / ".config"), TERM="xterm-256color")
    config_dir = root / ("Library/Application Support" if sys.platform == "darwin" else ".config") / "independent-harness"
    config_dir.mkdir(parents=True)
    (config_dir / "config.json").write_text(json.dumps({"profiles": {"fast": {"model": "fixture-fast"}}}))
    master, slave = pty.openpty()
    # A fixed 80x24 terminal, matched by the screen emulator.
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 24, 80, 0, 0))
    original = termios.tcgetattr(slave)
    app = subprocess.Popen([binary], cwd=project, env=env, stdin=slave, stdout=slave, stderr=slave)
    screen = Screen(24, 80)

    class ScreenBytes(bytearray):
        """Terminal output that also updates the emulated screen."""

        def extend(self, data):
            super().extend(data)
            screen.feed(data)

    output = ScreenBytes()

    def wait_for(text, start=0):
        # The renderer redraws only changed cells, so text can be on screen
        # without appearing contiguously in the bytes. A screen match counts
        # only for text that was not already shown when the wait began.
        needle = text.decode("utf-8", "replace")
        was_shown = needle in screen.text()
        found = lambda: text in output[start:] or (not was_shown and needle in screen.text())
        deadline = time.monotonic() + 8
        while not found() and time.monotonic() < deadline:
            if select.select([master], [], [], .1)[0]:
                output.extend(os.read(master, 65536))
        assert found(), (text, bytes(output[-900:]))

    def wait_finished(journal, offset):
        # Fast turns can finish without repainting an already visible status.
        deadline = time.monotonic() + 8
        while time.monotonic() < deadline:
            added = journal.read_bytes()[offset:]
            complete = added[:added.rfind(b"\n")+1].splitlines()
            if complete:
                last = json.loads(complete[-1])
                if last["type"] == "run_finished":
                    assert last["data"]["status"] == "completed", last
                    return
            if select.select([master], [], [], .05)[0]:
                output.extend(os.read(master, 65536))
        raise AssertionError("New turn did not finish")

    try:
        wait_for(b"GoHarness")
        wait_for(b"Chat setup required")
        assert not requests
        os.write(master, b"Should stay blocked\r")
        time.sleep(.1)
        assert not requests
        os.write(master, b"\x07")
        wait_for(b"GoHarness  /  Status")
        assert not requests
        os.write(master, b"p")
        wait_for(b"Provider setup")
        os.write(master, f"http://127.0.0.1:{server.server_port}/proxy/v1".encode() + b"\t\tsetup-fixture\t\t\t" + b"incorrect-test-key")
        wait_for("••••".encode())
        os.write(master, b"\x13")
        wait_for(b"Agent: build")
        wait_for(b"Ctrl+P commands")
        time.sleep(.1)
        pos = len(output)
        os.write(master, b"\x0c")
        wait_for(b"Should stay blocked", pos)
        os.write(master, b"\x15Helo from setup\x01\x1b[C\x1b[Cl\x05\r")
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
        os.write(master, b"\x15Retry after fixing key\r")
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
        os.write(master, b"Verify selected model\x10")
        wait_for(b"GoHarness  /  Commands", pos)
        os.write(master, b"/models\r")
        wait_for(b"Select model", pos)
        os.write(master, b"fast")
        time.sleep(.05)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"Agent: build  \xc2\xb7  fixture-fast", pos)
        time.sleep(.1)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"completed", pos)
        assert prompts[-1] == "Verify selected model", prompts
        assert models[-1] == "fixture-fast", models
        assert list(root.rglob("*.jsonl")) == [journal_before]
        pos = len(output)
        os.write(master, b"/agents\r")
        wait_for(b"Select agent", pos)
        os.write(master, b"plan")
        time.sleep(.05)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"Agent: plan", pos)
        time.sleep(.1)
        pos = len(output)
        os.write(master, b"Inspect in read-only mode\r")
        wait_for(b"completed", pos)
        assert not ({"shell", "edit_file", "write_file", "create_file", "apply_patch"} & toolsets[-1]), toolsets[-1]
        pos = len(output)
        os.write(master, b"\t")
        wait_for(b"Agent: build", pos)
        pos = len(output)
        os.write(master, b"/models\r")
        wait_for(b"Select model", pos)
        os.write(master, b"default")
        time.sleep(.05)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"Agent: build", pos)
        time.sleep(.1)
        pos = len(output)
        os.write(master, b"Verify default model\r")
        wait_for(b"completed", pos)
        assert models[-1] == "setup-fixture", models
        pos = len(output)
        os.write(master, b"/new\r")
        wait_for(b"Agent: build", pos)
        deadline = time.monotonic()+8
        while len(list(root.rglob("*.jsonl"))) < 2 and time.monotonic() < deadline:
            if select.select([master], [], [], .05)[0]:
                output.extend(os.read(master, 65536))
        assert len(list(root.rglob("*.jsonl"))) == 2, list(root.rglob("*.jsonl"))
        time.sleep(.1)
        pos = len(output)
        second_journal, = [p for p in root.rglob("*.jsonl") if p != journal_before]
        second_bytes = second_journal.read_bytes()
        os.write(master, b"/sessions\r")
        wait_for(b"GoHarness  /  Sessions", pos)
        os.write(master, b"Hello from setup")
        wait_for(b"Hello from setup", pos)
        time.sleep(.2)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"Agent: build", pos)
        time.sleep(.1)
        pos = len(output)
        os.write(master, b"Resumed journal verification\r")
        wait_for(b"completed", pos)
        assert b"Resumed journal verification" in journal_before.read_bytes()
        assert second_journal.read_bytes() == second_bytes
        assert len(list(root.rglob("*.jsonl"))) == 2
        before_preview = journal_before.read_bytes()
        preview = subprocess.run([binary, "sessions", "preview", journal_before.stem, "--session-dir", str(journal_before.parent)], cwd=project, env=env, capture_output=True, text=True, timeout=8)
        assert preview.returncode == 0, preview.stderr
        assert json.loads(preview.stdout)["title"] == "Hello from setup"
        assert journal_before.read_bytes() == before_preview
        pos = len(output)
        os.write(master, b"/sessions\r")
        wait_for(b"GoHarness  /  Sessions", pos)
        pos = len(output)
        os.write(master, b"\x1b")
        wait_for(b"Agent: build", pos)
        time.sleep(.1)
        assert journal_before.read_bytes() == before_preview
        pos = len(output)
        count = len(requests)
        before_timeline = journal_before.read_bytes()
        os.write(master, b"/timeline\r")
        wait_for(b"Timeline", pos)
        os.write(master, b"Verify selected model")
        time.sleep(.1)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"Prompt preview", pos)
        os.write(master, b"\x12")
        time.sleep(.15)
        assert len(requests) == count
        assert journal_before.read_bytes() == before_timeline
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"completed", pos)
        assert prompts[-1] == "Verify selected model", prompts
        time.sleep(.1)
        count = len(requests)
        pos = len(output)
        before_files = journal_before.read_bytes()
        os.write(master, b"Inspect @qf")
        wait_for(b"docs/query fixture.md", pos)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b'@"docs/query fixture.md"', pos)
        assert len(requests) == count
        assert journal_before.read_bytes() == before_files
        os.write(master, b"\r")
        wait_finished(journal_before, len(before_files))
        assert len(requests) == count+1
        # Choosing a file inserts only its path (no request, nothing shown);
        # sending attaches it like any @path, quoted because of the space.
        assert prompts[-1].startswith('Inspect @"docs/query fixture.md" '), prompts
        assert '<file path="docs/query fixture.md" bytes=' in prompts[-1], prompts
        assert b"FILE_BODY_MUST_STAY_LOCAL" not in output
        time.sleep(.1)
        count = len(requests)
        pos = len(output)
        before_status = journal_before.read_bytes()
        os.write(master, b"/status\r")
        wait_for(b"GoHarness  /  Status", pos)
        wait_for(b"Local key present", pos)
        os.write(master, b"\x1b[F")
        wait_for(b"PERMISSIONS", pos)
        os.write(master, b"r")
        time.sleep(.1)
        assert len(requests) == count
        pos = len(output)
        os.write(master, b"p")
        wait_for(b"Provider setup", pos)
        pos = len(output)
        os.write(master, b"\x1b")
        wait_for(b"Agent: build", pos)
        time.sleep(.1)
        assert journal_before.read_bytes() == before_status
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
        time.sleep(.1)
        # /connect offers the provider catalog; a preset shows its key page
        # and saves nothing until a key is checked.
        settings_before = (config_dir / "config.json").read_bytes()
        count = len(requests)
        pos = len(output)
        os.write(master, b"/connect\r")
        wait_for(b"Connect a provider", pos)
        wait_for(b"Anthropic (Claude API)", pos)
        os.write(master, b"anthropic")
        time.sleep(.1)
        os.write(master, b"\r")
        wait_for(b"platform.claude.com/settings/keys", pos)
        os.write(master, b"\x13")
        wait_for(b"paste an API key first", pos)
        pos = len(output)
        os.write(master, b"\x1b")
        wait_for(b"Agent: build", pos)
        time.sleep(.1)
        assert (config_dir / "config.json").read_bytes() == settings_before and len(requests) == count
        count = len(requests)
        before_export = journal_before.read_bytes()
        pos = len(output)
        os.write(master, b"/export\r")
        wait_for(b"Export transcript", pos)
        os.write(master, b"\x15local transcript.md")
        wait_for(b"local transcript.md", pos)
        os.write(master, b"\x1b")
        time.sleep(.15)
        export = project / "local transcript.md"
        assert not export.exists()
        pos = len(output)
        os.write(master, b"/export\r")
        wait_for(b"Export transcript", pos)
        os.write(master, b"\x15local transcript.md\r")
        wait_for(b"Transcript saved", pos)
        original_export = export.read_bytes()
        assert b"## User\n\nHello from setup" in original_export
        assert b"## Assistant\n\nSetup authenticated" in original_export
        assert stat.S_IMODE(export.stat().st_mode) == 0o600
        assert len(requests) == count and journal_before.read_bytes() == before_export
        os.write(master, b"\x1b")
        time.sleep(.15)
        pos = len(output)
        os.write(master, b"/export\r")
        wait_for(b"Export transcript", pos)
        os.write(master, b"\x15local transcript.md\r")
        wait_for(b"already exists; choose another name", pos)
        assert export.read_bytes() == original_export
        os.write(master, b"\x1b")
        time.sleep(.15)
        before_fork = journal_before.read_bytes()
        os.write(master, b"Unfinished source draft\x10")
        wait_for(b"GoHarness  /  Commands", pos)
        os.write(master, b"fork\r")
        wait_for(b"Full session", pos)
        os.write(master, b"\x1b")
        time.sleep(.15)
        pos = len(output)
        os.write(master, b"\x0c")
        wait_for(b"Unfinished source draft", pos)
        os.write(master, b"\x15/fork\r")
        pos = len(output)
        wait_for(b"Fork session", pos)
        os.write(master, b"Before turn 4")
        wait_for(b"Before turn 4", pos)
        assert len(list(root.rglob("*.jsonl"))) == 2
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"\x1b[?1049h", pos)
        wait_for(b"Verify selected model", output.index(b"\x1b[?1049h", pos))
        time.sleep(.1)
        branch, = [p for p in root.rglob("*.jsonl") if p not in (journal_before, second_journal)]
        branch_snapshot = branch.read_bytes()
        state = [json.loads(line) for line in branch_snapshot.splitlines()]
        assert state[1]["type"] == "session_forked"
        assert state[1]["data"]["source_session_id"] == journal_before.stem
        assert state[1]["data"]["before_turn"] == 4
        branch_prompts = [r["data"]["content"] for r in state if r["type"] == "message" and r["data"]["role"] == "user"]
        assert branch_prompts == ["Hello from setup", "Retry after fixing key", "Retry after fixing key"], branch_prompts
        assert len(requests) == count and journal_before.read_bytes() == before_fork
        os.write(master, b"\x15Branch experiment\r")
        wait_finished(branch, len(branch_snapshot))
        time.sleep(.1)
        assert len(requests) == count + 1 and prompts[-1] == "Branch experiment", prompts
        assert journal_before.read_bytes() == before_fork
        branch_offset = branch.stat().st_size
        pos = len(output)
        os.write(master, b"Compatibility fallback\r")
        wait_for(b"Compatible JSON received", pos)
        wait_finished(branch, branch_offset)
        assert prompts[-2:] == ["Compatibility fallback", "Compatibility fallback"]
        records = [json.loads(line) for line in branch.read_bytes()[branch_offset:].splitlines()]
        user_messages = [r for r in records if r["type"] == "message" and r["data"]["role"] == "user"]
        assert len(user_messages) == 1
        time.sleep(.1)
        os.write(master, b"\x04")
        assert app.wait(timeout=8) == 0
        cli_fork = subprocess.run(
            [binary, "sessions", "fork", journal_before.stem, "--session-dir",
             str(journal_before.parent), "--before-turn", "1"],
            cwd=project, env=env, capture_output=True, text=True, timeout=8)
        assert cli_fork.returncode == 0, cli_fork.stderr
        cli_branch = journal_before.parent / (json.loads(cli_fork.stdout)["id"] + ".jsonl")
        assert [json.loads(line)["type"] for line in cli_branch.read_bytes().splitlines()] == [
            "session_start", "session_forked"]
        assert journal_before.read_bytes() == before_fork
        cli_export = subprocess.run(
            [binary, "sessions", "export", journal_before.stem, "--session-dir",
             str(journal_before.parent), "--format", "markdown"],
            cwd=project, env=env, capture_output=True, timeout=8)
        assert cli_export.returncode == 0, cli_export.stderr
        assert cli_export.stdout == original_export
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
        print(json.dumps({"setup": "passed", "compatibility_fallback_verified": True, "provider_draft_recovery_verified": True, "markdown_export_verified": True, "fork_before_turn_verified": True, "file_completion_path_only_verified": True, "timeline_readonly_and_explicit_recall_verified": True, "status_readonly_verified": True, "status_provider_shortcut_verified": True, "session_browser_resume_verified": True, "session_preview_readonly": True, "model_picker_verified": True, "plan_tools_verified": True, "agent_cycle_verified": True, "new_session_verified": True, "chat_opens_before_setup": True, "unconfigured_chat_makes_no_requests": True, "key_masked": True, "cursor_editing_verified": True, "history_submission_verified": True, "authentication_error_recoverable": True, "session_preserved": True, "saved_key_used_by_cli": True, "settings_reachable_from_chat": True, "terminal_restored": True}))
    finally:
        if app.poll() is None:
            app.kill()
            app.wait()
        os.close(master)
        os.close(slave)
        server.shutdown()
