package smpp

const (
	SessionActive int32 = iota
	SessionClosed
)

const (
	CloseByError    = "error"
	CloseByPdu      = "pdu"
	CloseByExplicit = "explicit"
)

const (
	SubmitterSys int8 = iota
	SubmitterUser
)
