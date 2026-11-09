package report

import (
	"fmt"

	"example.com/task/units"
)

// Line formats a temperature reading.
func Line(city string, celsius float64) string {
	return fmt.Sprintf("%s: %.0fF", city, units.CToF(celsius))
}
