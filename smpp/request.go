package smpp

import (
	"github.com/linxGnu/gosmpp/pdu"
)

type Request struct {
	// submitted pdu
	Pdu pdu.PDU

	// trace info
	TraceData any
	SessionId string
	SystemId  string
	SubmitAt  int64

	// mark the submitter
	submitter int8
}
