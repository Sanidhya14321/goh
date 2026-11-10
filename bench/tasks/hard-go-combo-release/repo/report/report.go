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
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	io.WriteString(w, "sku,name,price,quantity\n")
	for _, it := range sorted {
		io.WriteString(w, it.SKU+","+it.Name+","+it.Price.String()+","+strconv.Itoa(it.Quantity)+"\n")
	}
	return nil
}
