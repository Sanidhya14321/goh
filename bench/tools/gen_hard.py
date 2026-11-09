#!/usr/bin/env python3
"""Generate the hard benchmark tasks from two correct mini-codebases.

Each task copies a correct base project, injects a realistic bug (exact text
replacement), and describes only the symptom. hidden/ holds tests for the full
intended behaviour; solution/ holds the original (correct) files. Because
the base is correct, every task is valid by construction; `goh-bench
--dry-run` confirms it. Re-run after editing: python3 bench/tools/gen_hard.py
"""
import json, os, shutil

ROOT = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "tasks")

GO = {
"go.mod": "module example.com/invsvc\n\ngo 1.22\n",
"model/model.go": '''package model

import "fmt"

// Cents is an amount of money in integer cents; floats are never used.
type Cents int64

func (c Cents) String() string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s%d.%02d", sign, c/100, c%100)
}

// Item is a stock-keeping unit.
type Item struct {
	SKU      string
	Name     string
	Price    Cents
	Quantity int
	Region   string
}
''',
"events/bus.go": '''package events

import "sync"

// Event is a change notification.
type Event struct {
	Kind string // "updated", "deleted"
	SKU  string
}

// Bus delivers events to subscribers in subscription order.
type Bus struct {
	mu   sync.Mutex
	subs []func(Event)
}

// Subscribe registers fn for all later events.
func (b *Bus) Subscribe(fn func(Event)) {
	b.mu.Lock()
	b.subs = append(b.subs, fn)
	b.mu.Unlock()
}

// Publish delivers e to every subscriber. Subscribers may publish again.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	subs := append([]func(Event){}, b.subs...)
	b.mu.Unlock()
	for _, fn := range subs {
		fn(e)
	}
}
''',
"store/store.go": '''package store

import (
	"errors"
	"sort"
	"sync"

	"example.com/invsvc/events"
	"example.com/invsvc/model"
)

var ErrNotFound = errors.New("item not found")

// Store is an in-memory item store that publishes change events.
type Store struct {
	mu    sync.RWMutex
	items map[string]model.Item
	bus   *events.Bus
}

func New(bus *events.Bus) *Store { return &Store{items: map[string]model.Item{}, bus: bus} }

func (s *Store) Get(sku string) (model.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	it, ok := s.items[sku]
	if !ok {
		return model.Item{}, ErrNotFound
	}
	return it, nil
}

func (s *Store) Put(it model.Item) {
	s.mu.Lock()
	s.items[it.SKU] = it
	s.mu.Unlock()
	s.bus.Publish(events.Event{Kind: "updated", SKU: it.SKU})
}

func (s *Store) Delete(sku string) {
	s.mu.Lock()
	delete(s.items, sku)
	s.mu.Unlock()
	s.bus.Publish(events.Event{Kind: "deleted", SKU: sku})
}

// List returns items sorted by SKU.
func (s *Store) List() []model.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Item, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SKU < out[j].SKU })
	return out
}

// Apply runs all updates atomically: if any update fails, none is applied
// and no events are published.
func (s *Store) Apply(updates []func(map[string]model.Item) error) error {
	s.mu.Lock()
	draft := make(map[string]model.Item, len(s.items))
	for k, v := range s.items {
		draft[k] = v
	}
	for _, u := range updates {
		if err := u(draft); err != nil {
			s.mu.Unlock()
			return err
		}
	}
	var changed []string
	for k, v := range draft {
		if old, ok := s.items[k]; !ok || old != v {
			changed = append(changed, k)
		}
	}
	s.items = draft
	s.mu.Unlock()
	sort.Strings(changed)
	for _, k := range changed {
		s.bus.Publish(events.Event{Kind: "updated", SKU: k})
	}
	return nil
}
''',
"cache/cache.go": '''package cache

import (
	"sync"
	"time"
)

// Clock lets tests control time.
type Clock func() time.Time

type entry struct {
	value   string
	expires time.Time
}

// TTL is a string cache whose entries expire after ttl.
type TTL struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   Clock
	items map[string]entry
}

func NewTTL(ttl time.Duration, now Clock) *TTL {
	return &TTL{ttl: ttl, now: now, items: map[string]entry{}}
}

func (c *TTL) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || !c.now().Before(e.expires) {
		delete(c.items, key)
		return "", false
	}
	return e.value, true
}

func (c *TTL) Set(key, value string) {
	c.mu.Lock()
	c.items[key] = entry{value, c.now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *TTL) Invalidate(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}
''',
"pricing/pricing.go": '''package pricing

import (
	"example.com/invsvc/model"
)

// TaxBasisPoints is the sales tax per region in basis points (1/100 %).
var TaxBasisPoints = map[string]int64{"CA": 725, "NY": 400, "TX": 625, "OR": 0}

// Tax returns the tax on amount for region, rounded half up to the cent.
func Tax(amount model.Cents, region string) model.Cents {
	bp := TaxBasisPoints[region]
	return model.Cents((int64(amount)*bp + 5000) / 10000)
}

// Line is one invoice line.
type Line struct {
	Item model.Item
	Qty  int
}

// Totals is an invoice summary. Total always equals the sum of LineTotals.
type Totals struct {
	LineTotals []model.Cents
	Subtotal   model.Cents
	Tax        model.Cents
	Total      model.Cents
}

// Invoice computes per-line totals (price*qty plus that line's tax) and
// sums them, so the invoice total matches the printed lines exactly.
func Invoice(lines []Line) Totals {
	var t Totals
	for _, l := range lines {
		net := l.Item.Price * model.Cents(l.Qty)
		tax := Tax(net, l.Item.Region)
		t.LineTotals = append(t.LineTotals, net+tax)
		t.Subtotal += net
		t.Tax += tax
	}
	t.Total = t.Subtotal + t.Tax
	return t
}
''',
"service/service.go": '''package service

import (
	"example.com/invsvc/cache"
	"example.com/invsvc/events"
	"example.com/invsvc/store"
)

// Service wires the store, price cache and events together.
type Service struct {
	Store *store.Store
	Cache *cache.TTL
	Bus   *events.Bus
}

// New builds a service. Cached price strings are invalidated whenever an
// item is updated or deleted, so callers never see a stale price.
func New(bus *events.Bus, c *cache.TTL) *Service {
	s := &Service{Store: store.New(bus), Cache: c, Bus: bus}
	bus.Subscribe(func(e events.Event) { c.Invalidate(e.SKU) })
	return s
}

// PriceText returns the item's price as text, using the cache.
func (s *Service) PriceText(sku string) (string, error) {
	if v, ok := s.Cache.Get(sku); ok {
		return v, nil
	}
	it, err := s.Store.Get(sku)
	if err != nil {
		return "", err
	}
	v := it.Price.String()
	s.Cache.Set(sku, v)
	return v, nil
}
''',
"api/api.go": '''package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"example.com/invsvc/model"
	"example.com/invsvc/store"
)

// Page is one page of items. Next is the SKU to continue after ("" at end).
type Page struct {
	Items []model.Item `json:"items"`
	Next  string       `json:"next"`
}

// ListItems pages through items in SKU order: ?after=SKU&limit=N. The cursor
// is the last SKU returned, so deleting items between requests never skips
// or repeats an item.
func ListItems(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}
		after := r.URL.Query().Get("after")
		var page Page
		for _, it := range s.List() {
			if after != "" && it.SKU <= after {
				continue
			}
			if len(page.Items) == limit {
				page.Next = page.Items[len(page.Items)-1].SKU
				break
			}
			page.Items = append(page.Items, it)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	}
}
''',
"report/report.go": '''package report

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"

	"example.com/invsvc/model"
)

// WriteCSV writes items sorted by name then SKU, with a header row. Fields
// are CSV-quoted as needed (names may contain commas and quotes).
func WriteCSV(w io.Writer, items []model.Item) error {
	sorted := append([]model.Item(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].SKU < sorted[j].SKU
	})
	cw := csv.NewWriter(w)
	cw.Write([]string{"sku", "name", "price", "quantity"})
	for _, it := range sorted {
		cw.Write([]string{it.SKU, it.Name, it.Price.String(), strconv.Itoa(it.Quantity)})
	}
	cw.Flush()
	return cw.Error()
}
''',
"store/store_test.go": '''package store

import (
	"testing"

	"example.com/invsvc/events"
	"example.com/invsvc/model"
)

func TestPutGet(t *testing.T) {
	s := New(&events.Bus{})
	s.Put(model.Item{SKU: "A1", Name: "Bolt", Price: 125})
	if it, err := s.Get("A1"); err != nil || it.Price != 125 {
		t.Fatal(it, err)
	}
}
''',
"pricing/pricing_test.go": '''package pricing

import "testing"

func TestTaxOregon(t *testing.T) {
	if Tax(1000, "OR") != 0 {
		t.Fatal("OR has no sales tax")
	}
}
''',
}

PY = {
"billing/__init__.py": "",
"billing/money.py": '''from decimal import Decimal, ROUND_HALF_UP

CENT = Decimal("0.01")


def to_money(value):
    """Return value as a Decimal rounded half up to the cent."""
    return Decimal(str(value)).quantize(CENT, rounding=ROUND_HALF_UP)
''',
"billing/periods.py": '''import calendar
from datetime import date


def days_in_month(d):
    return calendar.monthrange(d.year, d.month)[1]


def remaining_fraction(change_day):
    """Fraction of change_day's month remaining, counting change_day itself."""
    total = days_in_month(change_day)
    remaining = total - change_day.day + 1
    return remaining / total
''',
"billing/proration.py": '''from decimal import Decimal

from billing.money import to_money
from billing.periods import days_in_month


def prorated_upgrade(old_price, new_price, change_day):
    """Charge for upgrading mid-month: the price difference for the days
    remaining in the month, counting the change day, using that month's
    actual length (February and leap years included)."""
    total = days_in_month(change_day)
    remaining = total - change_day.day + 1
    diff = Decimal(str(new_price)) - Decimal(str(old_price))
    return to_money(diff * remaining / total)
''',
"billing/rates.py": '''from billing.money import to_money


class RateCache:
    """Caches exchange rates per (currency, day); rates change daily."""

    def __init__(self, fetch):
        self._fetch = fetch
        self._cache = {}

    def rate(self, currency, day):
        key = (currency, day)
        if key not in self._cache:
            self._cache[key] = self._fetch(currency, day)
        return self._cache[key]


def convert(amount, currency, day, cache):
    """Convert amount in currency to USD at that day's rate."""
    if currency == "USD":
        return to_money(amount)
    return to_money(amount * cache.rate(currency, day))
''',
"billing/payments.py": '''class TimeoutError_(Exception):
    pass


class PaymentError(Exception):
    pass


def charge_with_retry(gateway, invoice_id, amount, attempts=3):
    """Charge once. A timeout may mean the charge went through, so before each
    retry the gateway is asked whether invoice_id was already charged; a
    charge is never sent twice. Other errors are raised immediately."""
    for attempt in range(attempts):
        try:
            return gateway.charge(invoice_id, amount)
        except TimeoutError_:
            existing = gateway.lookup(invoice_id)
            if existing is not None:
                return existing
            if attempt == attempts - 1:
                raise PaymentError("payment timed out")
''',
"billing/usage.py": '''from collections import defaultdict
from datetime import datetime, timezone


def daily_totals(lines):
    """Sum usage per UTC day from lines like
    '2026-03-01T23:30:00-05:00 api 12' (ISO time with offset, meter, units).
    Days are UTC calendar days."""
    totals = defaultdict(int)
    for line in lines:
        line = line.strip()
        if not line:
            continue
        stamp, meter, units = line.split()
        when = datetime.fromisoformat(stamp).astimezone(timezone.utc)
        totals[(when.date().isoformat(), meter)] += int(units)
    return dict(totals)
''',
"tests/__init__.py": "",
"tests/test_basic.py": '''import unittest
from decimal import Decimal
from billing.money import to_money


class Money(unittest.TestCase):
    def test_round(self):
        self.assertEqual(to_money(2.675), Decimal("2.68"))


if __name__ == "__main__":
    unittest.main()
''',
}

GT = "go test ./..."
PT = "python3 -m unittest discover -v"

TASKS = [
 dict(id="hard-go-stale-price", base=GO,
  prompt="Customers report seeing an old price for up to a minute after we change an item's price or delete it; the price API should always reflect the latest change. Find and fix the cause.",
  mutate=[("service/service.go", "\tbus.Subscribe(func(e events.Event) { c.Invalidate(e.SKU) })\n", "")],
  hidden={"service/service_test.go": '''package service

import (
	"testing"
	"time"

	"example.com/invsvc/cache"
	"example.com/invsvc/events"
	"example.com/invsvc/model"
)

func TestPriceUpdatesImmediately(t *testing.T) {
	now := time.Unix(0, 0)
	s := New(&events.Bus{}, cache.NewTTL(time.Minute, func() time.Time { return now }))
	s.Store.Put(model.Item{SKU: "X", Price: 100})
	if v, _ := s.PriceText("X"); v != "1.00" {
		t.Fatal(v)
	}
	s.Store.Put(model.Item{SKU: "X", Price: 250})
	if v, _ := s.PriceText("X"); v != "2.50" {
		t.Fatalf("stale price after update: %s", v)
	}
	s.Store.Delete("X")
	if _, err := s.PriceText("X"); err == nil {
		t.Fatal("deleted item still priced")
	}
}
'''}, f2p="go test ./service/", p2p=GT),
 dict(id="hard-go-partial-import", base=GO,
  prompt="After a bulk import fails halfway (for example on a bad row), some items are already changed and subscribers were told about changes that should never have happened. Imports must be all-or-nothing. Fix it.",
  mutate=[("store/store.go", '''	s.mu.Lock()
	draft := make(map[string]model.Item, len(s.items))
	for k, v := range s.items {
		draft[k] = v
	}
	for _, u := range updates {''', '''	s.mu.Lock()
	draft := s.items
	for _, u := range updates {''')],
  hidden={"store/apply_test.go": '''package store

import (
	"errors"
	"testing"

	"example.com/invsvc/events"
	"example.com/invsvc/model"
)

func TestApplyIsAtomic(t *testing.T) {
	bus := &events.Bus{}
	var seen []string
	bus.Subscribe(func(e events.Event) { seen = append(seen, e.SKU) })
	s := New(bus)
	s.Put(model.Item{SKU: "A", Quantity: 1})
	seen = nil
	err := s.Apply([]func(map[string]model.Item) error{
		func(m map[string]model.Item) error { it := m["A"]; it.Quantity = 99; m["A"] = it; return nil },
		func(m map[string]model.Item) error { m["B"] = model.Item{SKU: "B"}; return nil },
		func(map[string]model.Item) error { return errors.New("bad row") },
	})
	if err == nil {
		t.Fatal("error lost")
	}
	if it, _ := s.Get("A"); it.Quantity != 1 {
		t.Fatalf("partial update applied: %+v", it)
	}
	if _, err := s.Get("B"); err == nil {
		t.Fatal("partial insert applied")
	}
	if len(seen) != 0 {
		t.Fatalf("events published for a failed import: %v", seen)
	}
	if err := s.Apply([]func(map[string]model.Item) error{func(m map[string]model.Item) error { m["C"] = model.Item{SKU: "C"}; return nil }}); err != nil || len(seen) != 1 || seen[0] != "C" {
		t.Fatalf("successful apply: %v %v", err, seen)
	}
}
'''}, f2p="go test ./store/", p2p=GT),
 dict(id="hard-go-invoice-cents", base=GO,
  prompt="Finance says some invoices don't add up: the invoice total differs from the sum of the printed line totals by a cent, mostly on multi-region orders. Every line must show its own tax-inclusive total and the invoice total must equal their sum. Fix the calculation.",
  mutate=[("pricing/pricing.go", '''		tax := Tax(net, l.Item.Region)
		t.LineTotals = append(t.LineTotals, net+tax)
		t.Subtotal += net
		t.Tax += tax
	}
	t.Total = t.Subtotal + t.Tax''', '''		t.LineTotals = append(t.LineTotals, net+Tax(net, l.Item.Region))
		t.Subtotal += net
	}
	region := "OR"
	if len(lines) > 0 {
		region = lines[0].Item.Region
	}
	t.Tax = Tax(t.Subtotal, region)
	t.Total = t.Subtotal + t.Tax''')],
  hidden={"pricing/invoice_test.go": '''package pricing

import (
	"testing"

	"example.com/invsvc/model"
)

func TestInvoiceTotalsMatchLines(t *testing.T) {
	lines := []Line{
		{Item: model.Item{SKU: "a", Price: 333, Region: "CA"}, Qty: 3},
		{Item: model.Item{SKU: "b", Price: 1999, Region: "NY"}, Qty: 1},
		{Item: model.Item{SKU: "c", Price: 101, Region: "TX"}, Qty: 7},
	}
	got := Invoice(lines)
	var sum model.Cents
	for _, lt := range got.LineTotals {
		sum += lt
	}
	if got.Total != sum || got.Total != got.Subtotal+got.Tax {
		t.Fatalf("%+v (lines sum %v)", got, sum)
	}
	if got.Tax != Tax(999, "CA")+Tax(1999, "NY")+Tax(707, "TX") {
		t.Fatalf("tax must be per line and region: %v", got.Tax)
	}
}

func TestTaxRoundsHalfUp(t *testing.T) {
	if Tax(200, "NY") != 8 || Tax(7, "CA") != 1 || Tax(6, "CA") != 0 {
		t.Fatal(Tax(200, "NY"), Tax(7, "CA"), Tax(6, "CA"))
	}
}
'''}, f2p="go test ./pricing/", p2p=GT),
 dict(id="hard-go-pagination-skip", base=GO,
  prompt="Clients paging through GET /items (using the `next` value as `after`) sometimes miss items or get the same item twice, especially when items are deleted while they page. Fix paging so every remaining item is returned exactly once.",
  mutate=[("api/api.go", '''			if len(page.Items) == limit {
				page.Next = page.Items[len(page.Items)-1].SKU
				break
			}
			page.Items = append(page.Items, it)''', '''			page.Items = append(page.Items, it)
			if len(page.Items) == limit {
				page.Next = it.SKU
				break
			}''')],
  hidden={"api/api_test.go": '''package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"example.com/invsvc/events"
	"example.com/invsvc/model"
	"example.com/invsvc/store"
)

func get(t *testing.T, s *store.Store, after string, limit int) Page {
	rec := httptest.NewRecorder()
	ListItems(s)(rec, httptest.NewRequest("GET", fmt.Sprintf("/items?after=%s&limit=%d", after, limit), nil))
	var p Page
	json.Unmarshal(rec.Body.Bytes(), &p)
	return p
}

func TestPagingVisitsEveryItemOnce(t *testing.T) {
	s := store.New(&events.Bus{})
	for i := 0; i < 7; i++ {
		s.Put(model.Item{SKU: fmt.Sprintf("S%02d", i)})
	}
	seen := map[string]int{}
	after := ""
	for pages := 0; pages < 10; pages++ {
		p := get(t, s, after, 3)
		for _, it := range p.Items {
			seen[it.SKU]++
		}
		if pages == 0 {
			s.Delete("S01") // already returned
			s.Delete("S04") // not yet returned
		}
		if p.Next == "" {
			break
		}
		after = p.Next
	}
	for i := 0; i < 7; i++ {
		sku := fmt.Sprintf("S%02d", i)
		want := 1
		if sku == "S04" {
			want = 0
		}
		if seen[sku] != want {
			t.Fatalf("%s seen %d times: %v", sku, seen[sku], seen)
		}
	}
}

func TestLastPageHasNoNext(t *testing.T) {
	s := store.New(&events.Bus{})
	for i := 0; i < 3; i++ {
		s.Put(model.Item{SKU: fmt.Sprintf("S%d", i)})
	}
	if p := get(t, s, "", 3); len(p.Items) != 3 || p.Next != "" {
		t.Fatalf("%+v", p)
	}
}
'''}, f2p="go test ./api/", p2p=GT),
 dict(id="hard-go-bus-deadlock", base=GO,
  prompt="The service sometimes hangs forever when a restock handler reacts to an item update by updating another item. Event handlers must be allowed to publish further events. Fix the hang without dropping or reordering events.",
  mutate=[("events/bus.go", '''	b.mu.Lock()
	subs := append([]func(Event){}, b.subs...)
	b.mu.Unlock()
	for _, fn := range subs {
		fn(e)
	}''', '''	b.mu.Lock()
	defer b.mu.Unlock()
	for _, fn := range b.subs {
		fn(e)
	}''')],
  hidden={"events/bus_test.go": '''package events

import (
	"testing"
	"time"
)

func TestHandlersMayPublish(t *testing.T) {
	var b Bus
	var got []string
	b.Subscribe(func(e Event) {
		got = append(got, e.SKU)
		if e.SKU == "A" {
			b.Publish(Event{Kind: "updated", SKU: "B"})
		}
	})
	done := make(chan struct{})
	go func() { b.Publish(Event{Kind: "updated", SKU: "A"}); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish deadlocked")
	}
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Fatal(got)
	}
}

func TestSubscribeDuringPublish(t *testing.T) {
	var b Bus
	calls := 0
	b.Subscribe(func(Event) { calls++; b.Subscribe(func(Event) { calls += 10 }) })
	done := make(chan struct{})
	go func() { b.Publish(Event{SKU: "x"}); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Subscribe inside a handler deadlocked")
	}
}
'''}, f2p="go test -race ./events/", p2p=GT),
 dict(id="hard-go-report-order", base=GO,
  prompt="The inventory CSV export is broken: rows with commas in the product name split into extra columns in Excel, and the row order changes between runs for items with the same name. Rows must be sorted by name then SKU and fields properly quoted.",
  mutate=[("report/report.go", '''	sorted := append([]model.Item(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].SKU < sorted[j].SKU
	})
	cw := csv.NewWriter(w)
	cw.Write([]string{"sku", "name", "price", "quantity"})
	for _, it := range sorted {
		cw.Write([]string{it.SKU, it.Name, it.Price.String(), strconv.Itoa(it.Quantity)})
	}
	cw.Flush()
	return cw.Error()''', '''	sorted := append([]model.Item(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	io.WriteString(w, "sku,name,price,quantity\\n")
	for _, it := range sorted {
		io.WriteString(w, it.SKU+","+it.Name+","+it.Price.String()+","+strconv.Itoa(it.Quantity)+"\\n")
	}
	return nil''')],
  hidden={"report/report_test.go": '''package report

import (
	"encoding/csv"
	"strings"
	"testing"

	"example.com/invsvc/model"
)

func TestCSVQuotingAndOrder(t *testing.T) {
	items := []model.Item{
		{SKU: "Z9", Name: "Bolt", Price: 5, Quantity: 1},
		{SKU: "A1", Name: "Nut, hex \\"M6\\"", Price: 12, Quantity: 40},
		{SKU: "B2", Name: "Bolt", Price: 7, Quantity: 2},
		{SKU: "A0", Name: "Bolt", Price: 9, Quantity: 3},
	}
	var b strings.Builder
	if err := WriteCSV(&b, items); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(b.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"sku", "name", "price", "quantity"}, {"A0", "Bolt", "0.09", "3"}, {"B2", "Bolt", "0.07", "2"}, {"Z9", "Bolt", "0.05", "1"}, {"A1", "Nut, hex \\"M6\\"", "0.12", "40"}}
	if len(rows) != len(want) {
		t.Fatal(rows)
	}
	for i := range want {
		if strings.Join(rows[i], "|") != strings.Join(want[i], "|") {
			t.Fatalf("row %d: %v want %v", i, rows[i], want[i])
		}
	}
}
'''}, f2p="go test ./report/", p2p=GT),
 dict(id="hard-py-feb-proration", base=PY,
  prompt="Upgrade charges are wrong for customers who change plans mid-month: in February (especially in leap years) and on the last day of a month the prorated amount is off. The charge should be the price difference for the remaining days of that month, counting the change day itself. Fix it.",
  mutate=[("billing/proration.py", '''    total = days_in_month(change_day)
    remaining = total - change_day.day + 1''', '''    total = 30
    remaining = total - change_day.day''')],
  hidden={"tests/test_proration.py": '''import unittest
from datetime import date
from decimal import Decimal
from billing.proration import prorated_upgrade


class Proration(unittest.TestCase):
    def test_leap_february(self):
        self.assertEqual(prorated_upgrade(10, 39, date(2028, 2, 15)), Decimal("15.00"))

    def test_last_day(self):
        self.assertEqual(prorated_upgrade(0, 31, date(2026, 1, 31)), Decimal("1.00"))

    def test_first_day_full_month(self):
        self.assertEqual(prorated_upgrade(5, 25, date(2026, 4, 1)), Decimal("20.00"))

    def test_rounding(self):
        self.assertEqual(prorated_upgrade(0, 10, date(2026, 3, 10)), Decimal("7.10"))


if __name__ == "__main__":
    unittest.main()
'''}, f2p="python3 -m unittest tests.test_proration -v", p2p=PT),
 dict(id="hard-py-stale-rates", base=PY,
  prompt="Invoices in foreign currencies converted later in the month use the wrong exchange rate: a rate seen early in a run seems to stick for every later day. Each conversion must use that day's rate. Fix it.",
  mutate=[("billing/rates.py", '''        key = (currency, day)''', '''        key = currency''')],
  hidden={"tests/test_rates.py": '''import unittest
from datetime import date
from decimal import Decimal
from billing.rates import RateCache, convert


class Rates(unittest.TestCase):
    def test_rate_per_day(self):
        calls = []
        table = {("EUR", date(2026, 3, 1)): Decimal("1.10"), ("EUR", date(2026, 3, 2)): Decimal("1.20")}

        def fetch(cur, day):
            calls.append((cur, day))
            return table[(cur, day)]
        cache = RateCache(fetch)
        self.assertEqual(convert(Decimal("100"), "EUR", date(2026, 3, 1), cache), Decimal("110.00"))
        self.assertEqual(convert(Decimal("100"), "EUR", date(2026, 3, 2), cache), Decimal("120.00"))
        self.assertEqual(convert(Decimal("50"), "EUR", date(2026, 3, 1), cache), Decimal("55.00"))
        self.assertEqual(len(calls), 2, "rates must still be cached per day")

    def test_usd_untouched(self):
        self.assertEqual(convert(Decimal("3.333"), "USD", date(2026, 3, 1), RateCache(None)), Decimal("3.33"))


if __name__ == "__main__":
    unittest.main()
'''}, f2p="python3 -m unittest tests.test_rates -v", p2p=PT),
 dict(id="hard-py-double-charge", base=PY,
  prompt="Some customers were charged twice for one invoice during a payment-gateway slowdown. Retrying after a timeout must never create a second charge for the same invoice, and non-timeout errors should not be retried. Fix the payment flow.",
  mutate=[("billing/payments.py", '''        except TimeoutError_:
            existing = gateway.lookup(invoice_id)
            if existing is not None:
                return existing
            if attempt == attempts - 1:''', '''        except Exception:
            if attempt == attempts - 1:''')],
  hidden={"tests/test_payments.py": '''import unittest
from billing.payments import charge_with_retry, TimeoutError_, PaymentError


class Gateway:
    def __init__(self, timeouts_after_success=0, fail_with=None):
        self.charges = []
        self.timeouts = timeouts_after_success
        self.fail_with = fail_with

    def charge(self, invoice, amount):
        if self.fail_with:
            raise self.fail_with
        self.charges.append(invoice)
        if self.timeouts:
            self.timeouts -= 1
            raise TimeoutError_()
        return "ch_%d" % len(self.charges)

    def lookup(self, invoice):
        return "ch_found" if invoice in self.charges else None


class Payments(unittest.TestCase):
    def test_timeout_after_success_is_not_recharged(self):
        g = Gateway(timeouts_after_success=1)
        self.assertEqual(charge_with_retry(g, "inv1", 10), "ch_found")
        self.assertEqual(g.charges, ["inv1"])

    def test_other_errors_not_retried(self):
        g = Gateway(fail_with=ValueError("card declined"))
        with self.assertRaises(ValueError):
            charge_with_retry(g, "inv2", 10)

    def test_plain_success(self):
        self.assertEqual(charge_with_retry(Gateway(), "inv3", 5), "ch_1")


if __name__ == "__main__":
    unittest.main()
'''}, f2p="python3 -m unittest tests.test_payments -v", p2p=PT),
 dict(id="hard-py-usage-days", base=PY,
  prompt="Daily usage totals don't match the provider's dashboard: usage logged late in the evening in US time zones lands on the wrong day. Days are UTC calendar days. Fix the aggregation.",
  mutate=[("billing/usage.py", '''        when = datetime.fromisoformat(stamp).astimezone(timezone.utc)
        totals[(when.date().isoformat(), meter)] += int(units)''', '''        totals[(stamp[:10], meter)] += int(units)''')],
  hidden={"tests/test_usage.py": '''import unittest
from billing.usage import daily_totals


class Usage(unittest.TestCase):
    def test_utc_days(self):
        lines = [
            "2026-03-01T23:30:00-05:00 api 12",
            "2026-03-02T01:00:00+00:00 api 3",
            "2026-03-02T09:00:00+09:00 api 5",
            "",
            "2026-03-01T10:00:00+00:00 storage 7",
        ]
        self.assertEqual(daily_totals(lines), {
            ("2026-03-02", "api"): 20,
            ("2026-03-01", "storage"): 7,
        })


if __name__ == "__main__":
    unittest.main()
'''}, f2p="python3 -m unittest tests.test_usage -v", p2p=PT),
]


def write(base, path, body):
    p = os.path.join(base, path)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    open(p, "w").write(body)


def main():
    for t in TASKS:
        d = os.path.join(ROOT, t["id"])
        shutil.rmtree(d, ignore_errors=True)
        files = dict(t["base"])
        solution = {}
        for path, good, bad in t["mutate"]:
            assert files[path].count(good) == 1, (t["id"], path, good[:60])
            solution[path] = files[path]
            files[path] = files[path].replace(good, bad)
        for path, body in files.items():
            write(os.path.join(d, "repo"), path, body)
        for path, body in t["hidden"].items():
            write(os.path.join(d, "hidden"), path, body)
        for path, body in solution.items():
            write(os.path.join(d, "solution"), path, body)
        meta = {"id": t["id"], "prompt": t["prompt"], "fail_to_pass": t["f2p"], "pass_to_pass": t["p2p"],
                "max_run_seconds": 1200, "max_tool_calls": 150}
        open(os.path.join(d, "task.json"), "w").write(json.dumps(meta, indent=2) + "\n")
        print("wrote", t["id"])


if __name__ == "__main__":
    main()
