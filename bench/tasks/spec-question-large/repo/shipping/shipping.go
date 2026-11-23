package shipping

// freeShippingFrom is the order subtotal, in cents, from which shipping is free.
const freeShippingFrom = 10000

// ShippingFee returns the shipping charge in cents for an order subtotal.
func ShippingFee(subtotal int) int {
	if subtotal >= freeShippingFrom {
		return 0
	}
	return 799
}
