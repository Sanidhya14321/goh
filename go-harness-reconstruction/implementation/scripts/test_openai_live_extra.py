#!/usr/bin/env python3
"""Live checks for features first verified offline: image input, reasoning
effort, trailing task state and plan handoff. Uses the saved ChatGPT sign-in;
makes real requests. Run only with the user's approval."""
import argparse, json, pathlib, struct, subprocess, tempfile, zlib


def png(width, height, rgb):
    raw = b"".join(b"\x00" + bytes(rgb) * width for _ in range(height))
    def chunk(kind, data):
        return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data) & 0xffffffff)
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)) + chunk(b"IDAT", zlib.compress(raw)) + chunk(b"IEND", b"")


def main(binary):
    results = {}
    with tempfile.TemporaryDirectory(prefix="goh-live-extra-") as d:
        root = pathlib.Path(d)
        work = root / "project"; work.mkdir()
        (work / "red.png").write_bytes(png(64, 64, (220, 20, 20)))
        (work / "note.txt").write_text("alpha\n")

        def run(prompt, config, extra=()):
            cfg = root / "config.json"
            cfg.write_text(json.dumps({"max_turns": 8, "context_window": 128000, **config}))
            p = subprocess.run([str(binary), "run", "--config", str(cfg), "--protocol", "openai_chatgpt",
                                "--session-dir", str(root / "sessions"), "--format", "json", *extra, "--prompt", prompt],
                               cwd=work, capture_output=True, text=True, timeout=180)
            events = [json.loads(l) for l in p.stdout.splitlines() if l.strip()]
            finished = [e["data"] for e in events if e["type"] == "run_finished"]
            text = "".join(e["data"]["message"]["content"] for e in events if e["type"] == "assistant_completed")
            tools = [e["data"]["name"] for e in events if e["type"] == "tool_started"]
            return (finished[-1]["status"] if finished else "none"), text, tools, p.stderr[-400:]

        status, text, _, err = run("What single colour fills the image @red.png? Answer with one word.", {})
        results["image_input"] = {"status": status, "answer": text.strip()[:80], "ok": status == "completed" and "red" in text.lower()}

        status, text, _, err = run("Reply exactly GOH_EFFORT_OK.", {"reasoning_effort": "low"})
        results["reasoning_effort_low"] = {"status": status, "ok": status == "completed" and "GOH_EFFORT_OK" in text, "stderr": err if status != "completed" else ""}

        status, text, tools, err = run("Use todowrite to record two tasks ('read note', 'report'), then read note.txt and tell me its first word.", {}, ["--allow", "todowrite"])
        results["trailing_task_state"] = {"status": status, "tools": tools, "ok": status == "completed" and "todowrite" in tools and "alpha" in text.lower(), "stderr": err if status != "completed" else ""}

        status, text, tools, err = run("Plan how to append the word beta to note.txt in two short steps, then call plan_exit with that plan.", {}, ["--agent", "plan", "--allow", "plan_exit"])
        results["plan_handoff"] = {"status": status, "tools": tools, "ok": status == "completed" and "plan_exit" in tools, "stderr": err if status != "completed" else ""}
    print(json.dumps(results, indent=2))
    return 0 if all(r["ok"] for r in results.values()) else 1


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", type=pathlib.Path, required=True)
    raise SystemExit(main(parser.parse_args().binary.resolve()))
