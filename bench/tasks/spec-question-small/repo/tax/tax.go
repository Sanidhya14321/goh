package tax

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
