package report

import (
	"encoding/csv"
	"strings"
	"testing"

	"example.com/invsvc/model"
)

func TestCSVQuotingAndOrder(t *testing.T) {
	items := []model.Item{
		{SKU: "Z9", Name: "Bolt", Price: 5, Quantity: 1},
		{SKU: "A1", Name: "Nut, hex \"M6\"", Price: 12, Quantity: 40},
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
	want := [][]string{{"sku", "name", "price", "quantity"}, {"A0", "Bolt", "0.09", "3"}, {"B2", "Bolt", "0.07", "2"}, {"Z9", "Bolt", "0.05", "1"}, {"A1", "Nut, hex \"M6\"", "0.12", "40"}}
	if len(rows) != len(want) {
		t.Fatal(rows)
	}
	for i := range want {
		if strings.Join(rows[i], "|") != strings.Join(want[i], "|") {
			t.Fatalf("row %d: %v want %v", i, rows[i], want[i])
		}
	}
}
