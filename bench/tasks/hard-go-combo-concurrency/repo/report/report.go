package report

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
