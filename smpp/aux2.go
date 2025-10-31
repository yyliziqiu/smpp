package smpp

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
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

func PrintPdu(tag string, systemId string, p pdu.PDU) {
	if p != nil {
		bs, _ := json.MarshalIndent(p, "", "  ")
		fmt.Printf("[%s:%s:%T] %s\n\n", tag, systemId, p, string(bs))
	}
}
