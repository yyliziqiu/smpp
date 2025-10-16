package smpp

import (
	"github.com/linxGnu/gosmpp/pdu"
)

// Request transmit pdu
type Request struct {
	Pdu pdu.PDU

	// trace info
	TraceData any
	SessionId string
	SystemId  string
	SubmitAt  int64

	// mark the submitter
	submitter int8
}
