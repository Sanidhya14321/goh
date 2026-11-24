#!/usr/bin/env python3
"""Fill each task's gold_files: the files a fix needs.

usage: bench/tools/gold_files.py [TASKS_DIR]   (default bench/tasks)

A bundled task's gold files are those its reference solution/ adds or
changes relative to repo/; a real-project task's are its source_files (the
upstream fix). Tasks graded only by their answer have no solution and are
skipped. The list is benchmark telemetry (retrieval recall, time to the first
gold file) and is never shown to the agent. Run it again after regenerating
tasks; the line is inserted after "fail_to_pass" (or replaced), leaving the
rest of task.json as written.
"""
import filecmp, json, os, pathlib, re, sys

root = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "bench/tasks")
updated = skipped = 0
for task_dir in sorted(p for p in root.iterdir() if (p / "task.json").is_file()):
    path = task_dir / "task.json"
    text = path.read_text()
    task = json.loads(text)
    gold = list(task.get("source_files") or [])
    solution, repo = task_dir / "solution", task_dir / "repo"
    if not gold and solution.is_dir():
        for f in sorted(solution.rglob("*")):
            if not f.is_file():
                continue
            rel = f.relative_to(solution).as_posix()
            original = repo / rel
            if not original.is_file() or not filecmp.cmp(f, original, shallow=False):
                gold.append(rel)
    if not gold:
        skipped += 1
        continue
    line = '  "gold_files": ' + json.dumps(gold) + ","
    if '"gold_files"' in text:
        text = re.sub(r'^\s*"gold_files":.*$', line, text, count=1, flags=re.M)
    else:
        text, n = re.subn(r'^(\s*"fail_to_pass":.*,)$', lambda m: m.group(1) + "\n" + line, text, count=1, flags=re.M)
        if n == 0:
            print("no fail_to_pass line, not changed:", task_dir.name)
            continue
    json.loads(text)  # still valid
    if text != path.read_text():
        path.write_text(text)
        updated += 1
print(f"gold_files: {updated} tasks updated, {skipped} without a solution or source files")
