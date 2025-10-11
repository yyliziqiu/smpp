package smpp

import (
	"github.com/linxGnu/gosmpp/pdu"
)

// Request transmit pdu
type Request struct {
	Pdu pdu.PDU

	// trace info
	SessionId string
	MessageId string
	SystemId  string
	CreateAt  int64
	SubmitAt  int64

	// mark the submitter
	submitter int8
}

// Response will be created when received response of transmit pdu or error occurred.
// The Pdu will be nil if the Err is not nil.
type Response struct {
	Request *Request
	Pdu     pdu.PDU
	Err     error
}

func NewResponse(request *Request, p pdu.PDU, err error) *Response {
	return &Response{
		Request: request,
		Pdu:     p,
		Err:     err,
	}
}
