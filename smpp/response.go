package smpp

import (
	"github.com/linxGnu/gosmpp/pdu"
)

// Response will be created when received response of transmit pdu or error occurred.
type Response struct {
	// indicate this is a failed response, that didn't receive the response pdu
	Error error

	// this field will be nil if Error is not nil.
	Pdu pdu.PDU

	// the request of the response
	Request *Request
}

func NewResponse(req *Request, p pdu.PDU, err error) *Response {
	return &Response{Error: err, Pdu: p, Request: req}
}

func (t *Response) TraceData() any {
	return t.Request.TraceData
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
