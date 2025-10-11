package smpp

import (
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

// ============ Session ============

var _store = NewSessionStore()

func GetSession(id string) *Session {
	return _store.GetSession(id)
}

func GetSessions() map[string]*Session {
	return _store.GetSessions()
}

func CountSessions() int {
	return _store.CountSessions()
}

// ============ Message ============

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

func Gsm7bitMessage(s string) pdu.ShortMessage {
	sm, _ := pdu.NewShortMessageWithEncoding(s, data.GSM7BIT)
	return sm
}

func Ucs2Message(s string) pdu.ShortMessage {
	sm, _ := pdu.NewShortMessageWithEncoding(s, data.UCS2)
	return sm
}

func BinaryMessage(s []byte) pdu.ShortMessage {
	sm, _ := pdu.NewBinaryShortMessageWithEncoding(s, data.BINARY8BIT2)
	return sm
}
