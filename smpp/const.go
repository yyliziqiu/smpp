package smpp

const (
	ConnectionDialed int32 = iota
	ConnectionClosed
)

const (
	SessionDialing = "dialing"
	SessionActive  = "active"
	SessionClosed  = "closed"
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
