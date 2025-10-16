package smpp

import (
	"github.com/linxGnu/gosmpp/pdu"
)

// Response will be created when received response of transmit pdu or error occurred.
// The Pdu will be nil if the Error is not nil.
type Response struct {
	Request *Request
	Pdu     pdu.PDU
	Error   error
}

func NewResponse(request *Request, p pdu.PDU, err error) *Response {
	return &Response{
		Request: request,
		Pdu:     p,
		Error:   err,
	}
}
