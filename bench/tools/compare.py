#!/usr/bin/env python3
"""Compare goh-bench runs task by task.

usage: bench/tools/compare.py RESULTS_DIR [RESULTS_DIR ...]

Each argument is a goh-bench --out folder (or one run folder in it); a run
with several arms counts as one column per arm. For every task present,
prints per-attempt averages side by side: solved, false completions, model
requests, tool calls, input tokens (all purposes and child sessions, from the
journals), cached share, output tokens, seconds and, for tasks with
gold_files, how many of the files the fix needs the agent read (recall).
Use it after each change: the same tasks, the same attempts, before and
after.
"""
import glob, json, os, sys


def load(path):
    files = glob.glob(os.path.join(path, "results.jsonl")) or glob.glob(os.path.join(path, "*", "results.jsonl"))
    rows = [json.loads(line) for f in files for line in open(f) if line.strip()]
    return [r for r in rows if r.get("scored")]


def per_task(rows):
    out = {}
    for r in rows:
        a = out.setdefault(r["task"], {"n": 0, "solved": 0, "false": 0, "requests": 0, "tools": 0, "input": 0, "cached": 0, "output": 0, "seconds": 0.0, "recall": 0.0, "graded": 0})
        usage = r.get("usage_by_purpose") or {}
        a["n"] += 1
        a["solved"] += bool(r.get("resolved"))
        a["false"] += bool(r.get("false_completion"))
        if r.get("retrieval"):
            a["recall"] += r["retrieval"].get("recall", 0)
            a["graded"] += 1
        a["requests"] += r.get("requests", 0)
        a["tools"] += r.get("tool_calls", 0)
        a["input"] += sum(u.get("prompt_tokens", 0) for u in usage.values()) or r.get("prompt_tokens", 0)
        a["cached"] += sum(u.get("cache_read_tokens", 0) for u in usage.values()) or r.get("cache_read_tokens", 0)
        a["output"] += sum(u.get("completion_tokens", 0) for u in usage.values()) or r.get("output_tokens", 0)
        a["seconds"] += r.get("seconds", 0)
    return out


def main():
    runs = []
    for p in sys.argv[1:]:
        rows = load(p)
        arms = sorted({r.get("arm", "") for r in rows})
        name = os.path.basename(os.path.normpath(p))
        for arm in arms:
            label = f"{name}:{arm}" if len(arms) > 1 else name
            runs.append((label, per_task([r for r in rows if r.get("arm", "") == arm])))
    if not runs:
        sys.exit(__doc__)
    tasks = sorted({t for _, r in runs for t in r})
    names = " | ".join(n for n, _ in runs)
    print(f"Runs: {names}\n")
    print("| Task | Solved | False done | Requests | Tool calls | Input tokens | Cached | Output tokens | Seconds | Gold recall |")
    print("|---|---|---|---|---|---|---|---|---|---|")
    totals = [{"n": 0, "solved": 0, "false": 0, "requests": 0, "tools": 0, "input": 0, "cached": 0, "output": 0, "seconds": 0.0, "recall": 0.0, "graded": 0} for _ in runs]
    for t in tasks:
        cells = {k: [] for k in ("solved", "false", "requests", "tools", "input", "cached", "output", "seconds", "recall")}
        for i, (_, r) in enumerate(runs):
            a = r.get(t)
            if not a:
                for k in cells:
                    cells[k].append("-")
                continue
            for k in totals[i]:
                totals[i][k] += a[k]
            n = a["n"]
            cells["solved"].append(f'{a["solved"]}/{n}')
            cells["false"].append(str(a["false"]))
            cells["recall"].append(f'{100 * a["recall"] / a["graded"]:.0f}%' if a["graded"] else "-")
            cells["requests"].append(f'{a["requests"] / n:.1f}')
            cells["tools"].append(f'{a["tools"] / n:.1f}')
            cells["input"].append(f'{a["input"] / n:,.0f}')
            cells["cached"].append(f'{100 * a["cached"] / max(1, a["input"]):.0f}%')
            cells["output"].append(f'{a["output"] / n:,.0f}')
            cells["seconds"].append(f'{a["seconds"] / n:.0f}')
        print(f"| {t} | " + " | ".join(" → ".join(v) for v in cells.values()) + " |")
    row = []
    for k in ("solved", "false", "requests", "tools", "input", "cached", "output", "seconds", "recall"):
        vals = []
        for tot in totals:
            if k == "solved":
                vals.append(f'{tot["solved"]}/{tot["n"]}')
            elif k == "false":
                vals.append(str(tot["false"]))
            elif k == "recall":
                vals.append(f'{100 * tot["recall"] / tot["graded"]:.0f}%' if tot["graded"] else "-")
            elif k == "cached":
                vals.append(f'{100 * tot["cached"] / max(1, tot["input"]):.0f}%')
            else:
                vals.append(f'{tot[k] / max(1, tot["n"]):,.1f}' if k in ("requests", "tools") else f'{tot[k] / max(1, tot["n"]):,.0f}')
        row.append(" → ".join(vals))
    print("| **All (per attempt)** | " + " | ".join(row) + " |")


if __name__ == "__main__":
    main()
