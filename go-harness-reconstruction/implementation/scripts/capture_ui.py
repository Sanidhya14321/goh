#!/usr/bin/env python3
"""Capture real PTY frames and compare first prompt latency on this host.

Run with a verification-only Python environment containing pyte and Pillow.
Does not send prompts or install providers in OpenCode.
"""
import codecs
import fcntl
import hashlib
import json
import os
import pathlib
import platform
import pty
import select
import statistics
import struct
import socket
import subprocess
import tempfile
import termios
import time

import pyte
from PIL import Image, ImageDraw, ImageFont

base = pathlib.Path(__file__).resolve().parents[1]
evidence = base.parent / "research/evidence"
font = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", 16)
cols, rows = 120, 36

def screenshot(screen, name):
    image = Image.new("RGB", (cols * 10 + 40, rows * 23 + 40), "#0a0a0a")
    draw = ImageDraw.Draw(image)
    for y in range(rows):
        for x in range(cols):
            char = screen.buffer[y][x]
            bg = "#" + char.bg if len(char.bg) == 6 else "#0a0a0a"
            fg = "#" + char.fg if len(char.fg) == 6 else "#e6e6e6"
            box = (20 + x * 10, 20 + y * 23, 30 + x * 10, 43 + y * 23)
            draw.rectangle(box, fill=bg)
            draw.text((box[0], box[1]), char.data, font=font, fill=fg)
    image.save(evidence / (name + ".png"))
    (evidence / (name + ".txt")).write_text("\n".join(screen.display))

def capture(binary, label, trial):
    with tempfile.TemporaryDirectory(prefix="goh-ui-") as directory:
        root = pathlib.Path(directory)
        (root / "project").mkdir()
        env = {k: v for k, v in os.environ.items() if not k.startswith(("HARNESS_", "OPENCODE_", "OPENAI_", "ANTHROPIC_"))}
        env.pop("NO_COLOR", None)
        env.update(HOME=str(root), XDG_CONFIG_HOME=str(root / ".config"), XDG_DATA_HOME=str(root / ".data"),
                   XDG_STATE_HOME=str(root / ".state"), XDG_CACHE_HOME=str(root / ".cache"), TERM="xterm-256color", COLORTERM="truecolor")
        fixture = None
        provider_url = "http://127.0.0.1:9/v1"
        if trial == 0:
            with socket.socket() as sock:
                sock.bind(("127.0.0.1", 0))
                port = sock.getsockname()[1]
            def call(name, arguments):
                return {"calls": [{"id": "capture-" + name, "type": "function", "function": {"name": name, "arguments": json.dumps(arguments)}}]}
            fence = chr(96) * 3
            steps = [
                call("read_file", {"path": "note.txt"}),
                call("edit_file", {"path": "note.txt", "expected_sha256": hashlib.sha256(b"alpha\n").hexdigest(), "old_text": "alpha", "new_text": "beta"}),
                call("shell", {"command": 'test "$(cat note.txt)" = beta', "timeout_seconds": 5}),
                {"text": "## Verified change\n\nUpdated **note.txt** and checked its contents.\n\n" + fence + "go\nfunc ready() bool {\n    return true\n}\n" + fence + "\n\n| Check | Result |\n| :--- | ---: |\n| File edit | approved |\n| Verification | passed |\n\n" + fence + "json\n{\"ready\": true, \"count\": 42}\n" + fence},
            ]
            if label == "opencode":
                steps = steps[-1:] * 16
            script = root / "capture-script.json"
            script.write_text(json.dumps(steps))
            fixture = subprocess.Popen([str(base / "bin/scripted-model"), "--listen", f"127.0.0.1:{port}", "--script", str(script)], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            provider_url = f"http://127.0.0.1:{port}/v1"
            (root / "project/note.txt").write_text("alpha\n")
            (root / "project/docs").mkdir()
            (root / "project/docs/notes.md").write_text("# Notes\n")
            (root / "project/docs/notation.md").write_text("# Notation\n")
        if label == "goh":
            env.update(HARNESS_BASE_URL=provider_url, HARNESS_MODEL="fixture",
                       HARNESS_CONTEXT_WINDOW="131072", HARNESS_API_KEY_ENV="UNSET_UI_FIXTURE")
            marker = "Ctrl+P commands"
        else:
            env.update(OPENCODE_DISABLE_AUTOUPDATE="1", OPENCODE_DISABLE_MODELS_FETCH="1", OPENCODE_DISABLE_PROJECT_CONFIG="1",
                       OPENCODE_PURE="1", OPENCODE_AUTH_CONTENT="{}", OPENCODE_TEST_HOME=str(root),
                       OPENCODE_CONFIG_CONTENT=json.dumps({"model": "fixture/fixture", "provider": {"fixture": {
                           "npm": "@ai-sdk/openai-compatible", "name": "Local fixture", "options": {"baseURL": provider_url},
                           "models": {"fixture": {"name": "fixture", "limit": {"context": 131072, "output": 4096}}}
                       }}}))
            marker = "Ask anything"
        if label == "goh":
            config_dir = root / "Library/Application Support/independent-harness"
            config_dir.mkdir(parents=True)
            (config_dir / "config.json").write_text(json.dumps({"profiles": {"fast": {"model": "fixture-fast"}, "quality": {"model": "fixture-quality"}}, "agents": {"reviewer": {"mode": "plan", "description": "Read-only review", "instructions": "Review the project."}}}))
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
        screen = pyte.Screen(cols, rows)
        stream = pyte.Stream(screen)
        decoder = codecs.getincrementaldecoder("utf-8")("replace")
        raw = bytearray()
        start = time.perf_counter()
        app = subprocess.Popen([str(binary)], cwd=root / "project", env=env, stdin=slave, stdout=slave, stderr=slave, start_new_session=True)

        def drain(timeout, target=None):
            end = time.monotonic() + timeout
            while time.monotonic() < end:
                if app.poll() is not None:
                    raise RuntimeError(f"{label} exited early with {app.returncode}")
                if select.select([master], [], [], .01)[0]:
                    data = os.read(master, 65536)
                    raw.extend(data)
                    # Replies to capability probes make both terminals usable.
                    for query, reply in (
                        (b"\x1b[6n", b"\x1b[1;1R"),
                        (b"\x1b[c", b"\x1b[?1;2c"),
                        (b"\x1b]10;?", b"\x1b]10;rgb:eeee/eeee/eeee\x1b\\"),
                        (b"\x1b]11;?", b"\x1b]11;rgb:0a0a/0a0a/0a0a\x1b\\"),
                    ):
                        if query in data:
                            os.write(master, reply)
                    stream.feed(decoder.decode(data))
                    if target and target in "\n".join(screen.display):
                        return True
            return False

        try:
            if not drain(20, marker):
                raise RuntimeError(f"{label} prompt unavailable: " + "\n".join(screen.display))
            latency = (time.perf_counter() - start) * 1000
            drain(.2)
            rss = int(subprocess.check_output(["ps", "-o", "rss=", "-p", str(app.pid)], text=True).strip())
            if trial == 0:
                screenshot(screen, label + "-ui-home")
                os.write(master, b"Inspect @not")
                assert drain(8, "docs/notes.md"), "\n".join(screen.display)
                drain(.3)
                screenshot(screen, label + "-ui-file-completion")
                os.write(master, b"\x1b")
                drain(.2)
                os.write(master, b"\x15")
                drain(.2)
                if label == "goh":
                    os.write(master, b"\x10")
                    assert drain(3, "Search")
                    drain(.2)
                    screenshot(screen, "goh-ui-palette")
                    os.write(master, b"\x1b")
                    drain(.1)
                    os.write(master, b"Update note.txt and verify it\r")
                    assert drain(8, "APPROVAL: edit_file")
                    drain(.1)
                    screenshot(screen, "goh-ui-approval")
                    os.write(master, b"y")
                    assert drain(8, "APPROVAL: shell")
                    os.write(master, b"y")
                    assert drain(8, "Enter send")
                    drain(.1)
                    screenshot(screen, "goh-ui-session")
                    os.write(master, b"/pro")
                    drain(.1)
                    screenshot(screen, "goh-ui-slash")
                    os.write(master, b"\x15Review the generated code\x01Please ")
                    drain(.3)
                    screenshot(screen, "goh-ui-composer")
                    os.write(master, b"\x15/models\r")
                    assert drain(5, "Select model profile")
                    drain(.2)
                    screenshot(screen, "goh-ui-models")
                    os.write(master, b"\x1b")
                    drain(.2)
                    os.write(master, b"/agents\r")
                    assert drain(5, "Select agent")
                    drain(.2)
                    screenshot(screen, "goh-ui-agents")
                    os.write(master, b"\x1b")
                    drain(.2)
                    os.write(master, b"/sessions\r")
                    assert drain(5, "GoHarness  /  Sessions")
                    drain(.3)
                    screenshot(screen, "goh-ui-sessions")
                    os.write(master, b"\x1b")
                    drain(.2)
                    os.write(master, b"/status\r")
                    assert drain(5, "GoHarness  /  Status")
                    drain(.3)
                    screenshot(screen, "goh-ui-status")
                    os.write(master, b"\x1b")
                    drain(.2)
                    os.write(master, b"/timeline\r")
                    assert drain(5, "Timeline")
                    drain(.3)
                    screenshot(screen, "goh-ui-timeline")
                    os.write(master, b"\r")
                    assert drain(5, "Prompt preview")
                    drain(.3)
                    screenshot(screen, "goh-ui-prompt-preview")
                else:
                    os.write(master, b"Show a short code and table example\r")
                    assert drain(15, "Verification")
                    drain(.5)
                    screenshot(screen, "opencode-ui-session")
                    os.write(master, b"/models\r")
                    if not drain(.5, "Select model"):
                        os.write(master, b"\r")
                        assert drain(5, "Select model")
                    drain(.3)
                    screenshot(screen, "opencode-ui-models")
                    os.write(master, b"\x1b")
                    drain(.2)
                    os.write(master, b"/agents\r")
                    if not drain(.5, "Select agent"):
                        os.write(master, b"\r")
                        assert drain(5, "Select agent")
                    drain(.3)
                    screenshot(screen, "opencode-ui-agents")
                    os.write(master, b"\x1b")
                    drain(.2)
                    os.write(master, b"/sessions\r")
                    if not drain(.5, "Sessions"):
                        os.write(master, b"\r")
                        assert drain(5, "Sessions")
                    drain(.3)
                    screenshot(screen, "opencode-ui-sessions")
                    os.write(master, b"\x1b")
                    drain(.2)
                    os.write(master, b"/status\r")
                    if not drain(.5, "No MCP Servers"):
                        os.write(master, b"\r")
                        assert drain(5, "No MCP Servers")
                    drain(.3)
                    screenshot(screen, "opencode-ui-status")
                    os.write(master, b"\x1b")
                    drain(.2)
                    os.write(master, b"/timeline\r")
                    if not drain(.5, "Timeline"):
                        os.write(master, b"\r")
                        assert drain(5, "Timeline")
                    drain(.3)
                    screenshot(screen, "opencode-ui-timeline")
            return {"first_prompt_ms": round(latency, 2), "process_rss_kib": rss}
        finally:
            (evidence / f"{label}-ui-{trial}.ansi").write_bytes(raw)
            os.killpg(app.pid, 15)
            try:
                app.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(app.pid, 9)
                app.wait()
            os.close(master)
            os.close(slave)
            if fixture is not None:
                fixture.terminate()
                fixture.wait(timeout=5)

report = {"goh_version": subprocess.check_output([str(base / "bin/goh"), "version"], text=True).strip(), "opencode_version": subprocess.check_output([str(base.parent / "research/reference/opencode"), "--version"], text=True).strip(), "platform": platform.platform(), "terminal": "120x36 PTY", "scope": "first interactive home prompt, configured loopback provider, no inference", "trials": {}}
for label, binary in (("goh", base / "bin/goh"), ("opencode", base.parent / "research/reference/opencode")):
    report["trials"][label] = [capture(binary, label, i) for i in range(3)]
    report[label + "_median_ms"] = statistics.median(r["first_prompt_ms"] for r in report["trials"][label])
(evidence / "ui-startup-comparison.json").write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps(report, indent=2))
