package smpp

import (
	"github.com/linxGnu/gosmpp/pdu"
)

// RRequest received pdu
type RRequest struct {
	Session *Session
	Pdu     pdu.PDU
}

func NewRRequest(session *Session, p pdu.PDU) *RRequest {
	return &RRequest{Session: session, Pdu: p}
}

// TRequest transmitted pdu
type TRequest struct {
	SystemId  string
	SessionId string
	MessageId string
	CreateAt  int64
	SubmitAt  int64
	Pdu       pdu.PDU
	submitter int8
}

// TResponse will be created when received response of transmitted pdu or error occurred.
// The Pdu will be nil if the Err is not nil.
type TResponse struct {
	Session *Session
	Request *TRequest
	Pdu     pdu.PDU
	Error   error
}

func NewTResponse(session *Session, request *TRequest, p pdu.PDU, err error) *TResponse {
	return &TResponse{
		Session: session,
		Request: request,
		Pdu:     p,
		Error:   err,
	}
}
