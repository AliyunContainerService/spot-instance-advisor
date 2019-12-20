package pkg

import (
	"fmt"
)

func Decimal(value float64) string {
	valueStr := fmt.Sprintf("%.5f", value)
	return valueStr
}
