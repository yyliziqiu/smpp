package smpp

import (
	"github.com/linxGnu/gosmpp/pdu"
)

type Request struct {
	// submitted pdu
	Pdu pdu.PDU

	// trace info
	SessionId string
	SystemId  string
	SubmitAt  int64
	TraceData any

	// mark the submitter
	submitter int8
}

type Response struct {
	// the request of this response
	Request *Request

	// this field will be nil if Error is not nil
	Pdu pdu.PDU

	// indicate this is a failed response, that didn't receive the response pdu
	Error error
}

func NewResponse(req *Request, p pdu.PDU, err error) *Response {
	return &Response{Request: req, Pdu: p, Error: err}
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

func (t *Response) ErrorString() string {
	if t.Error == nil {
		return ""
	}
	return t.Error.Error()
}
