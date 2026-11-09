package units

// KMToMiles converts kilometres to miles.
func KMToMiles(km float64) float64 { return km / 1.609344 }

// CToF converts Celsius to Fahrenheit.
func CToF(c float64) float64 { return c*9/5 + 32 }
