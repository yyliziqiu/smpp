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

func (resp *Response) TraceData() any {
	return resp.Request.TraceData
}

func (resp *Response) TraceDataInt() (int, bool) {
	data, ok := resp.Request.TraceData.(int)
	return data, ok
}

func (resp *Response) TraceDataString() (string, bool) {
	data, ok := resp.Request.TraceData.(string)
	return data, ok
}

func (resp *Response) SessionId() string {
	return resp.Request.SessionId
}

func (resp *Response) SystemId() string {
	return resp.Request.SystemId
}

func (resp *Response) SubmitAt() int64 {
	return resp.Request.SubmitAt
}

func (resp *Response) Errors() string {
	if resp.Error == nil {
		return ""
	}
	return resp.Error.Error()
}
