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

func NewResponse(req *Request, p pdu.PDU, err error) *Response {
	return &Response{
		Request: req,
		Pdu:     p,
		Error:   err,
	}
}

func (t *Response) TraceData() any {
	return t.Request.TraceData
}

func (t *Response) SessionId() string {
	return t.Request.SessionId
}

func (t *Response) SystemId() string {
	return t.Request.SystemId
}

func (t *Response) SubmitAt() int64 {
	return t.Request.SubmitAt
}

func (t *Response) ErrorString() string {
	if t.Error == nil {
		return ""
	}
	return t.Error.Error()
}

func (t *Response) TraceInt() int {
	if i, ok := t.Request.TraceData.(int); ok {
		return i
	}
	return 0
}

func (t *Response) TraceString() string {
	if s, ok := t.Request.TraceData.(string); ok {
		return s
	}
	return ""
}
