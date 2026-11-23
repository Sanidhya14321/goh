#!/usr/bin/env python3
"""Generate the prompt-range ("spectrum") benchmark tasks.

The other suites are bug fixes of similar size. These tasks cover the range
of prompts an agent gets, so a harness change is judged on all of them:

  chat/tiny       "hello": one request, no tools
  chat/small      a general question that needs no project access
  question/small  a fact from a small project
  question/large  a fact inside one ~6,000-line file
  question/huge   a fact somewhere in a ~400-file project
  question/large  a long pasted log whose root cause is buried mid-way
  bugfix/large    a symptom-described bug in a ~6,000-line file (hidden test)

Question and chat tasks are graded by the final reply (answer patterns,
never shown to the agent) and must not change project files. Every task
states the efficient path (expected requests and tool calls); exceeding it
is reported, not failed. Content is generated deterministically. Re-run
after editing: python3 bench/tools/gen_spectrum.py, then
goh-bench --dry-run --only <ids> --record-identities for the bugfix task.
"""
import json, os, random, shutil

ROOT = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "tasks")
GOMOD = "module example.com/shop\n\ngo 1.22\n"

SMALL = {
    "go.mod": GOMOD,
    "README.md": "# shop\n\nA small order service.\n",
    "cart/cart.go": '''package cart

// Item is one line of an order, priced in cents.
type Item struct {
	SKU      string
	Quantity int
	Cents    int
}

// Subtotal is the sum of every line before shipping and tax.
func Subtotal(items []Item) int {
	total := 0
	for _, it := range items {
		total += it.Quantity * it.Cents
	}
	return total
}
''',
    "shipping/shipping.go": '''package shipping

// freeShippingFrom is the order subtotal, in cents, from which shipping is free.
const freeShippingFrom = 10000

// ShippingFee returns the shipping charge in cents for an order subtotal.
func ShippingFee(subtotal int) int {
	if subtotal >= freeShippingFrom {
		return 0
	}
	return 799
}
''',
    "tax/tax.go": '''package tax

// Rate returns the sales tax rate for a region, in basis points.
func Rate(region string) int {
	switch region {
	case "OR":
		return 0
	case "WA":
		return 650
	}
	return 500
}
''',
    "cart/cart_test.go": '''package cart

import "testing"

func TestSubtotal(t *testing.T) {
	if got := Subtotal([]Item{{"a", 2, 150}, {"b", 1, 99}}); got != 399 {
		t.Fatalf("subtotal %d", got)
	}
}
''',
}

VERBS = ["apply", "compute", "check", "format", "merge", "split", "round", "scale", "clamp", "match"]
NOUNS = ["discount", "invoice", "credit", "refund", "balance", "deposit", "charge", "payout", "voucher", "adjustment",
         "ledger", "statement", "accrual", "rebate", "surcharge", "coupon", "levy", "fee", "grant", "transfer"]


def filler(rng, n):
    """n plausible, distinct ledger helpers (about 14 lines each)."""
    out, seen = [], set()
    while len(out) < n:
        name = rng.choice(VERBS) + rng.choice(NOUNS).capitalize() + rng.choice(NOUNS).capitalize()
        if name in seen:
            continue
        seen.add(name)
        a, b, c = rng.randint(2, 97), rng.randint(100, 9000), rng.randint(1, 9)
        out.append(f'''// {name} adjusts an amount in cents by the {name[0].lower() + name[1:]} rule.
func {name}(cents int, units int) int {{
	if units <= 0 {{
		return cents
	}}
	adjusted := cents*{c} + units*{a}
	if adjusted > {b}*units {{
		adjusted = {b} * units
	}}
	if adjusted%{c + 1} == 0 {{
		adjusted++
	}}
	return adjusted
}}
''')
    return out


def ledger(bug):
    """A ~6,000-line ledger.go; the two functions the tasks need sit near the middle."""
    rng = random.Random(20261123)
    helpers = filler(rng, 420)
    special = '''// lateFeeCents is the fee for paying an invoice daysLate days after it was
// due: a flat charge per day, never more than the cap.
func lateFeeCents(daysLate int) int {
	const perDay, maxFee = 175, 4750
	if daysLate <= 0 {
		return 0
	}
	fee := daysLate * perDay
	if fee > maxFee {
		fee = maxFee
	}
	return fee
}

// ProrateCents charges for the days used of a billing period: usedDays of
// periodDays, both counted inclusively (cancelling on the last day of the
// period uses every day of it).
func ProrateCents(periodCents, usedDays, periodDays int) int {
	if periodDays <= 0 || usedDays <= 0 {
		return 0
	}
	if usedDays %s periodDays {
		return periodCents
	}
	return periodCents * %s / periodDays
}
''' % ((">", "(usedDays - 1)") if bug else (">=", "usedDays"))
    half = len(helpers) // 2
    body = "package ledger\n\n// Package ledger holds the billing arithmetic, all in integer cents.\n\n" + "\n".join(helpers[:half]) + "\n" + special + "\n" + "\n".join(helpers[half:])
    return body


def big_tree(rng):
    """About 400 small files in 20 packages; one sets the webhook retry backoff."""
    files = {"go.mod": GOMOD}
    pkgs = [f"svc{i:02d}" for i in range(20)]
    for p in pkgs:
        for j in range(20):
            name = f"{p}/{rng.choice(NOUNS)}_{j:02d}.go"
            files[name] = f'''package {p}

// Settings for the {p} {j} worker.
const (
	maxBatch{j} = {rng.randint(10, 500)}
	retryBackoffMillis{j} = {rng.randint(100, 900)}
	timeoutSeconds{j} = {rng.randint(5, 60)}
)

func Batch{j}(n int) int {{
	if n > maxBatch{j} {{
		return maxBatch{j}
	}}
	return n
}}
'''
    files["svc13/webhook_sender.go"] = '''package svc13

import "time"

// Defaults for delivering webhooks to customer endpoints.
const (
	webhookMaxAttempts = 6
	// webhookRetryBackoff is the wait before the first redelivery; it
	// doubles after each failed attempt.
	webhookRetryBackoff = 2750 * time.Millisecond
)

// NextWait is the wait before redelivery attempt n (1-based).
func NextWait(n int) time.Duration {
	return webhookRetryBackoff << (n - 1)
}
'''
    return files


def log_prompt():
    rng = random.Random(7)
    lines = []
    services = ["importer", "scheduler", "api-gw", "cache-warm", "billing-sync", "notifier"]
    for i in range(1300):
        t = f"2026-11-22T02:{(i // 60) % 60:02d}:{i % 60:02d}.{rng.randint(0, 999):03d}Z"
        s = rng.choice(services)
        msg = rng.choice(["batch committed rows=%d" % rng.randint(50, 900), "heartbeat ok", "cache hit ratio=0.%02d" % rng.randint(60, 99),
                          "request served status=200 ms=%d" % rng.randint(3, 80), "queue depth=%d" % rng.randint(0, 40),
                          "WARN slow query ms=%d table=orders" % rng.randint(500, 900)])
        if i == 611:
            lines.append(f"{t} ERROR [schema-migrator] migration 0042_tax_region failed: column \"tax_region\" of relation \"invoices\" does not exist; aborting at step 3/5")
            continue
        if i > 611 and i % 37 == 0:
            msg = "ERROR upstream connection reset by peer (retry %d)" % (i // 37)
            s = "importer"
        lines.append(f"{t} INFO [{s}] {msg}")
    return ("Our nightly import failed. Here is the full log from the run. What is the root cause? "
            "Name the component and the error.\n\n```\n" + "\n".join(lines) + "\n```")


def task(tid, meta, repo, hidden=None, solution=None):
    d = os.path.join(ROOT, tid)
    shutil.rmtree(d, ignore_errors=True)
    for sub, files in (("repo", repo), ("hidden", hidden or {}), ("solution", solution or {})):
        for rel, text in files.items():
            p = os.path.join(d, sub, rel)
            os.makedirs(os.path.dirname(p), exist_ok=True)
            with open(p, "w") as f:
                f.write(text)
    meta = dict(meta, id=tid)
    meta.setdefault("max_run_seconds", 600)
    meta.setdefault("max_tool_calls", 60)
    with open(os.path.join(d, "task.json"), "w") as f:
        json.dump(meta, f, indent=2)
        f.write("\n")


def answer(patterns, example, forbid=None):
    a = {"patterns": patterns, "read_only": True, "example": example}
    if forbid:
        a["forbid"] = forbid
    return a


task("spec-chat-hello", {"category": "chat", "size": "tiny", "prompt": "hello",
     "answer": answer([r"\b(hi|hello|hey|greetings)\b"], "Hi! What can I help you with?"),
     "expect": {"requests": 1, "tool_calls": 0}}, SMALL)

task("spec-chat-concept", {"category": "chat", "size": "small",
     "prompt": "Without looking at the project: in two sentences, when should I use a Go channel instead of a mutex?",
     "answer": answer([r"channel", r"mutex|lock"], "Use a channel to pass ownership of data or signal between goroutines; use a mutex to guard shared state that several goroutines read and write."),
     "expect": {"requests": 1, "tool_calls": 0}}, SMALL)

task("spec-question-small", {"category": "question", "size": "small",
     "prompt": "How much shipping does an order with a $40 subtotal pay, and from what subtotal is shipping free? Name the function that decides it.",
     "answer": answer([r"7\.99|799", r"\$?100(\.00)?\b|10,?000", r"ShippingFee"], "ShippingFee charges $7.99 (799 cents) below a $100.00 subtotal (10000 cents) and nothing from $100 up."),
     "expect": {"requests": 4, "tool_calls": 4}}, SMALL)

big_small = dict(SMALL)
big_small["ledger/ledger.go"] = ledger(bug=False)
task("spec-question-large", {"category": "question", "size": "large",
     "prompt": "In ledger/ledger.go, how is the late fee for an overdue invoice computed: what is charged per day late, and what is the maximum fee? Give both in cents.",
     "answer": answer([r"\b175\b", r"4,?750"], "lateFeeCents charges 175 cents per day late, capped at 4750 cents."),
     "expect": {"requests": 4, "tool_calls": 3}}, big_small)

buggy = dict(SMALL)
buggy["ledger/ledger.go"] = ledger(bug=True)
buggy["ledger/ledger_test.go"] = '''package ledger

import "testing"

func TestLateFee(t *testing.T) {
	if lateFeeCents(3) != 525 || lateFeeCents(100) != 4750 || lateFeeCents(0) != 0 {
		t.Fatal("late fee")
	}
}
'''
task("spec-bugfix-large", {"category": "bugfix", "size": "large",
     "prompt": "Prorated charges are a day short: a customer who used 15 days of a 30-day period pays for 14, and cancelling on the last day of the period does not charge the full price. Fix it.",
     "fail_to_pass": "go test ./ledger/", "pass_to_pass": "go test ./...",
     "expect": {"requests": 8, "tool_calls": 8}},
     buggy,
     hidden={"ledger/prorate_test.go": '''package ledger

import "testing"

func TestProrateFullPeriod(t *testing.T) {
	cases := []struct{ cents, used, days, want int }{
		{3000, 30, 30, 3000}, {3100, 31, 31, 3100}, {3000, 15, 30, 1500}, {3000, 0, 30, 0}, {3000, 45, 30, 3000},
	}
	for _, c := range cases {
		if got := ProrateCents(c.cents, c.used, c.days); got != c.want {
			t.Errorf("ProrateCents(%d, %d, %d) = %d, want %d", c.cents, c.used, c.days, got, c.want)
		}
	}
}
'''},
     solution={"ledger/ledger.go": ledger(bug=False)})

tree = big_tree(random.Random(42))
task("spec-question-huge", {"category": "question", "size": "huge",
     "prompt": "Which file sets the default wait before the first webhook redelivery, and what is that wait?",
     "answer": answer([r"webhook_sender\.go", r"2\.75\s*s|2,?750\s*(ms|milliseconds)"], "svc13/webhook_sender.go sets webhookRetryBackoff to 2750ms (2.75 s), doubling after each failed attempt."),
     "expect": {"requests": 4, "tool_calls": 4}}, tree)

task("spec-question-paste", {"category": "question", "size": "large", "prompt": log_prompt(),
     "answer": answer([r"schema[- ]migrat|migration", r"tax_region"], "The schema-migrator's migration 0042_tax_region failed because column tax_region does not exist on invoices; the later importer connection resets are fallout."),
     "expect": {"requests": 1, "tool_calls": 0}}, SMALL)

print("generated spectrum tasks in", ROOT)
