package smpp

import (
	"time"

	"github.com/linxGnu/gosmpp/pdu"
)

type Connection interface {
	SystemId() string
	BindType() pdu.BindingType
	SelfAddr() string
	PeerAddr() string
	Dial() error
	Read() (pdu.PDU, error)
	Write(pdu.PDU) (int, error)
	Close() error
	SetDeadline(time.Time) error
}
