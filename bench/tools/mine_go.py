#!/usr/bin/env python3
"""Mine benchmark tasks from real bug-fix commits in a Go repository.

A commit qualifies when it changes Go source and *_test.go files (small
source diff, no go.mod change). It is kept only if the package's tests, with
the fix commit's test files, FAIL on the parent with real test failures (not
compile errors) and PASS once the fix is applied; the repository's tests must
also pass at the fix. Output: bench/tasks/<id>/ with task.json (git source at
the parent commit), hidden/ (post-fix test files) and solution/ (post-fix
source files). Prompts are drafts from the commit subject: review them so
they describe the bug, not the fix.

usage: mine_go.py URL NAME [--max 3] [--scan 400]
"""
import json, os, re, shutil, subprocess, sys, tempfile

def run(cmd, cwd=None, timeout=600):
    p = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    return p.returncode, p.stdout + p.stderr

def git(repo, *a):
    code, out = run(["git", "-C", repo, *a])
    if code:
        raise RuntimeError(out)
    return out

def extract(repo, rev, dest):
    os.makedirs(dest, exist_ok=True)
    arch = subprocess.run(["git", "-C", repo, "archive", rev], capture_output=True, check=True).stdout
    subprocess.run(["tar", "-x", "-C", dest], input=arch, check=True)

def write_rev_files(repo, rev, paths, dest):
    for p in paths:
        code, out = run(["git", "-C", repo, "show", f"{rev}:{p}"])
        target = os.path.join(dest, p)
        if code:  # deleted in rev
            if os.path.exists(target):
                os.remove(target)
            continue
        os.makedirs(os.path.dirname(target), exist_ok=True)
        open(target, "w").write(out)

def main():
    url, name = sys.argv[1], sys.argv[2]
    maxn = int(sys.argv[sys.argv.index("--max") + 1]) if "--max" in sys.argv else 3
    scan = int(sys.argv[sys.argv.index("--scan") + 1]) if "--scan" in sys.argv else 400
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    cache = os.path.join(root, ".cache", "mine-" + name)
    if not os.path.isdir(cache):
        run(["git", "clone", "-q", url, cache], timeout=900)
    log = git(cache, "log", "--no-merges", "-n", str(scan), "-i", "-E", "--grep=fix|bug|wrong|incorrect|panic|race", "--format=%H%x00%P%x00%s")
    found = 0
    for line in log.splitlines():
        sha, parents, subject = line.split("\x00")
        if found >= maxn or " " in parents:
            continue
        files = []
        for row in git(cache, "diff", "--numstat", parents, sha).splitlines():
            add, dele, path = row.split("\t")
            files.append((path, int(add) if add != "-" else 999, int(dele) if dele != "-" else 999))
        tests = [f for f, _, _ in files if f.endswith("_test.go")]
        src = [(f, a, d) for f, a, d in files if f.endswith(".go") and not f.endswith("_test.go")]
        if not tests or not src or len(src) > 4 or any(f.startswith("go.") or "/go.mod" in f or "testdata" in f for f, _, _ in files):
            continue
        if sum(a + d for _, a, d in src) > 80:
            continue
        pkgs = sorted({"./" + (os.path.dirname(f) or ".") for f in tests})
        tmp = tempfile.mkdtemp()
        try:
            before = os.path.join(tmp, "before")
            extract(cache, parents, before)
            write_rev_files(cache, sha, tests, before)
            code, out = run(["go", "test", *pkgs], cwd=before)
            if code == 0 or "--- FAIL" not in out:
                continue  # passes already, or only a build failure
            after = os.path.join(tmp, "after")
            extract(cache, sha, after)
            code, _ = run(["go", "test", *pkgs], cwd=after)
            if code:
                continue
            code, _ = run(["go", "test", "./..."], cwd=after)
            if code:
                continue
            tid = f"go-{name}-{sha[:8]}"
            d = os.path.join(root, "tasks", tid)
            shutil.rmtree(d, ignore_errors=True)
            write_rev_files(cache, sha, tests, os.path.join(d, "hidden"))
            write_rev_files(cache, sha, [f for f, _, _ in src], os.path.join(d, "solution"))
            body = git(cache, "log", "-1", "--format=%b", sha).strip()
            task = {"id": tid, "git": {"url": url, "commit": parents},
                    "prompt": f"DRAFT (review): {subject}\n{body[:600]}",
                    "fail_to_pass": "go test " + " ".join(pkgs), "pass_to_pass": "go test ./...",
                    "max_run_seconds": 1200, "max_tool_calls": 150,
                    "source_commit": sha, "source_files": [f for f, _, _ in src]}
            os.makedirs(d, exist_ok=True)
            open(os.path.join(d, "task.json"), "w").write(json.dumps(task, indent=2) + "\n")
            print("KEPT", tid, subject)
            found += 1
        except Exception as e:  # noqa: BLE001 - skip unusable commits
            print("skip", sha[:8], e, file=sys.stderr)
        finally:
            shutil.rmtree(tmp, ignore_errors=True)
    print(f"{name}: kept {found}")

if __name__ == "__main__":
    main()
