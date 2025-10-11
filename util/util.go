package util

import (
	"fmt"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

var _logger *logrus.Logger

func SetLogger(logger *logrus.Logger) {
	_logger = logger
}

func LogDebug(s string, a ...any) {
	if _logger == nil {
		return
	}
	_logger.Debugf(s, a...)
}

func LogInfo(s string, a ...any) {
	if _logger == nil {
		return
	}
	_logger.Infof(s, a...)
}

func LogWarn(s string, a ...any) {
	if _logger == nil {
		return
	}
	_logger.Warnf(s, a...)
}

func LogError(s string, a ...any) {
	if _logger == nil {
		return
	}
	_logger.Errorf(s, a...)
}

func PrintMemory(tag string, gc bool) {
	if gc {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Printf("[memory:%s] alloc: %d KB\n", tag, memStats.Alloc/1024)
}
