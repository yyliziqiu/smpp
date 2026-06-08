package xuid

import (
	"errors"
	"strconv"
)

var ErrTimeBackForward = errors.New("time back forward")

var _generator = New(1)

func Get() string {
	return _generator.Get()
}

func GetOrFail() (string, error) {
	return _generator.GetOrFail()
}

var _padding = []string{
	"", "0", "00", "000", "0000", "00000", "000000", "0000000", "00000000", "000000000", "0000000000",
	"00000000000", "000000000000", "0000000000000", "00000000000000", "000000000000000", "0000000000000000",
}

func hex(n int64, l int) string {
	s := strconv.FormatInt(n, 16)
	if len(s) < l {
		s = _padding[l-len(s)] + s
	}
	return s
}
