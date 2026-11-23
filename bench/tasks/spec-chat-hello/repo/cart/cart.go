package cart

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
