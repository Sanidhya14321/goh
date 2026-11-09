package model

import "fmt"

// Cents is an amount of money in integer cents; floats are never used.
type Cents int64

func (c Cents) String() string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s%d.%02d", sign, c/100, c%100)
}

// Item is a stock-keeping unit.
type Item struct {
	SKU      string
	Name     string
	Price    Cents
	Quantity int
	Region   string
}
