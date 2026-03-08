package wautil

import (
	"fmt"
	"strings"
)

// NormalizeBillID normalizes various billID representations into a clean string.
func NormalizeBillID(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case *string:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(*v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

// NormalizeString safely dereferences *string to string.
func NormalizeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// NormalizeFloat safely dereferences *float64 to float64.
func NormalizeFloat(value *float64) float64 {
	if value == nil {
		return 0.0
	}
	return *value
}

