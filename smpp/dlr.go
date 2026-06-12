package smpp

const (
	DlrStatEnRoute       = "ENROUTE"
	DlrStatDelivered     = "DELIVRD"
	DlrStatExpired       = "EXPIRED"
	DlrStatDeleted       = "DELETED"
	DlrStatUndeliverable = "UNDELIV"
	DlrStatAccepted      = "ACCEPTD"
	DlrStatUnknown       = "UNKNOWN"
	DlrStatRejected      = "REJECTD"
)

type Dlr = Receipt

func ParseDlr(s string) (Dlr, error) {
	return ParseReceipt(s)
}

func BuildDlr(id string, sub int, dlvrd int, stat string, err int) Dlr {
	return BuildReceipt(id, sub, dlvrd, stat, err)
}
