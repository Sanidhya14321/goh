package pricing

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
