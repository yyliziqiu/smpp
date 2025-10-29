package smpp

const (
	ConnectionDialed int32 = iota
	ConnectionClosed
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
