package xconv

import (
	"strconv"
)

// S2I64 string to int64
func S2I64(s string) int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	return 0
}
