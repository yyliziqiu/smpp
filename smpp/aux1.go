package smpp

import (
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
	"github.com/sirupsen/logrus"
)

var (
	_tracer = NewTracer()

	_logger *logrus.Logger
)

// ============ Inner ============

func logDebug(s string, a ...any) {
	if _logger == nil {
		return
	}
	_logger.Debugf(s, a...)
}

func logInfo(s string, a ...any) {
	if _logger == nil {
		return
	}
	_logger.Infof(s, a...)
}

func logWarn(s string, a ...any) {
	if _logger == nil {
		return
	}
	_logger.Warnf(s, a...)
}

func logError(s string, a ...any) {
	if _logger == nil {
		return
	}
	_logger.Errorf(s, a...)
}

// ============ Outer ============

func SetLogger(logger *logrus.Logger) {
	_logger = logger
}

func GetSession(id string) *Session {
	return _tracer.GetSession(id)
}

func GetSessions() map[string]*Session {
	return _tracer.GetSessions()
}

func CountSession() int {
	return _tracer.CountSessions()
}

func Address(ton byte, npi byte, addr string) pdu.Address {
	ret := pdu.NewAddress()
	ret.SetTon(ton)
	ret.SetNpi(npi)
	_ = ret.SetAddress(addr)
	return ret
}

func Message(s string) pdu.ShortMessage {
	sm, _ := pdu.NewShortMessageWithEncoding(s, data.FindEncoding(s))
	return sm
}

func MessageInGsm7bit(s string) pdu.ShortMessage {
	sm, _ := pdu.NewShortMessageWithEncoding(s, data.GSM7BIT)
	return sm
}

func MessageInUcs2(s string) pdu.ShortMessage {
	sm, _ := pdu.NewShortMessageWithEncoding(s, data.UCS2)
	return sm
}

func MessageInBinary(s []byte) pdu.ShortMessage {
	sm, _ := pdu.NewBinaryShortMessageWithEncoding(s, data.BINARY8BIT2)
	return sm
}
