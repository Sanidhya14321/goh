#!/usr/bin/env python3
"""Run a small live Responses workflow after goh auth login."""
import argparse
import json
import os
import pty
import select
import pathlib
import subprocess
import tempfile
import time


def test(binary):
    with tempfile.TemporaryDirectory(prefix="goh-live-responses-") as directory:
        root = pathlib.Path(directory)
        work = root / "project"
        work.mkdir()
        note = work / "note.txt"
        note.write_text("alpha\n")
        config = root / "config.json"
        config.write_text(json.dumps({"max_turns": 8, "context_window": 32768}))
        common = ["--config", str(config), "--protocol", "openai_chatgpt",
                  "--session-dir", str(root / "sessions"), "--format", "json"]

        def run(action, prompt, options=()):
            process = subprocess.run(
                [str(binary), *action, *common, *options, "--prompt", prompt],
                cwd=work, capture_output=True, text=True, timeout=120)
            events = [json.loads(line) for line in process.stdout.splitlines()]
            finished = [e["data"] for e in events if e["type"] == "run_finished"]
            assert process.returncode == 0 and finished[-1]["status"] == "completed", (
                finished, process.stderr)
            return events

        def text(events):
            return "".join(e["data"]["message"]["content"] for e in events
                           if e["type"] == "assistant_completed")

        first = run(["run"], "Reply exactly GOH_NATIVE_OPENAI_OK. Do not call tools.")
        assert text(first).strip() == "GOH_NATIVE_OPENAI_OK"
        assert any(e["type"] == "assistant_updated" for e in first)
        session = first[0]["session_id"]
        second = run(["resume", session],
                     "Use read_file on note.txt. Use edit_file to replace alpha with beta using "
                     "the hash returned by read_file. Only modify note.txt. Read note.txt again "
                     "to verify. Do not run shell.", ["--allow", "edit_file"])
        assert note.read_text() == "beta\n"
        starts = {e["data"]["id"]: e["data"]["name"] for e in second if e["type"] == "tool_started"}
        completed = [starts[e["data"]["id"]] for e in second
                     if e["type"] == "tool_finished" and not e["data"]["result"].get("error", False)]
        edit = completed.index("edit_file")
        assert "read_file" in completed[:edit] and "read_file" in completed[edit+1:], completed
        assert any(e["type"] == "permission_resolved" for e in second)
        third = run(["resume", session],
                    "What did we change alpha to? Reply with the word only. Do not use tools.")
        assert text(third).strip().lower().rstrip(".") == "beta", text(third)
        verify_tui(binary, root, work, config)
        return {"version": subprocess.check_output([str(binary), "version"], text=True).strip(),
                "live_provider": "OpenAI Responses via GoHarness ChatGPT authorization",
                "model": first[0]["data"]["model"], "streamed_text": True,
                "permission_gated_read_edit_verify": True, "durable_resume": True,
                "successful_tools": completed, "tui_connect_and_send": True}


def verify_tui(binary, root, work, config):
    master, slave = pty.openpty()
    session_dir = root / "tui-sessions"
    app = subprocess.Popen(
        [str(binary), "chat", "--config", str(config), "--session-dir", str(session_dir)],
        cwd=work, stdin=slave, stdout=slave, stderr=slave,
        env={**os.environ, "TERM": "xterm-256color"})
    output = bytearray()

    def wait_for(marker, offset=0):
        deadline = time.monotonic() + 30
        while marker not in output[offset:] and time.monotonic() < deadline:
            if select.select([master], [], [], .1)[0]:
                output.extend(os.read(master, 65536))
        assert marker in output[offset:], (marker, bytes(output[-700:]))

    try:
        wait_for(b"Chat setup required")
        os.write(master, b"/connect\r")
        wait_for(b"Continue with ChatGPT")
        time.sleep(.15)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"Select ChatGPT model", pos)
        time.sleep(.15)
        pos = len(output)
        os.write(master, b"\r")
        wait_for(b"Agent: build", pos)
        time.sleep(.15)
        os.write(master, b"Reply exactly GOH_LIVE_TUI_OK. Do not call tools.\r")
        deadline = time.monotonic() + 45
        while time.monotonic() < deadline:
            if select.select([master], [], [], .1)[0]:
                output.extend(os.read(master, 65536))
            journals = list(session_dir.glob("*.jsonl"))
            if not journals:
                continue
            raw = journals[0].read_bytes()
            records = [json.loads(line) for line in raw[:raw.rfind(b"\n") + 1].splitlines()]
            if records and records[-1]["type"] == "run_finished":
                assert records[-1]["data"]["status"] == "completed", records[-1]
                assert any(r["type"] == "message" and r["data"]["role"] == "assistant"
                           and r["data"]["content"].strip() == "GOH_LIVE_TUI_OK" for r in records)
                break
        else:
            raise AssertionError("live terminal turn did not finish")
        time.sleep(.2)
        os.write(master, b"\x04")
        assert app.wait(timeout=8) == 0
    finally:
        if app.poll() is None:
            app.kill()
            app.wait()
        os.close(master)
        os.close(slave)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=pathlib.Path, default=pathlib.Path("bin/goh"))
    args = parser.parse_args()
    print(json.dumps(test(args.binary.resolve()), indent=2))
