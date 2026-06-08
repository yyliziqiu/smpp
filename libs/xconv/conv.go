package xconv

import (
	"strconv"
	"time"
)

// S2B string 转 bool
func S2B(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

// S2I string 转 int
func S2I(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// S2I64 string 转 int64
func S2I64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

// S2F64 string 转 float64
func S2F64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// S2T 字符串转时间戳
func S2T(s string) int64 {
	t, err := time.Parse(time.DateTime, s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// B2S bool 转 string
func B2S(b bool) string {
	return strconv.FormatBool(b)
}

// I2S int 转 string
func I2S(i int) string {
	return strconv.Itoa(i)
}

// I642S int64 转 string
func I642S(i int64) string {
	return strconv.FormatInt(i, 10)
}

// F642S float64 转 string
func F642S(f float64, prec int) string {
	return strconv.FormatFloat(f, 'f', prec, 64)
}

// T2S 时间戳转字符串
func T2S(t int64) string {
	return time.Unix(t, 0).Format(time.DateTime)
}
