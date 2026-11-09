package pricing

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
