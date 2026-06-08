package xtimer

import (
	"strconv"
	"time"
)

type Timer struct {
	start time.Time
}

func New() Timer {
	return Timer{
		start: time.Now(),
	}
}

func (t *Timer) Stop() string {
	return t.duration(time.Now().Sub(t.start))
}

var _units = []string{"ns", "us", "ms", "s"}

func (t *Timer) duration(d time.Duration) string {
	f := float64(d)
	i := 0
	for f > 1000 && i < len(_units)-1 {
		f = f / 1000
		i++
	}
	return strconv.FormatFloat(f, 'f', 2, 64) + _units[i]
}
