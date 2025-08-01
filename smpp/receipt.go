package smpp

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

const (
	ReceiptStatEnRoute       = "ENROUTE"
	ReceiptStatDelivered     = "DELIVRD"
	ReceiptStatExpired       = "EXPIRED"
	ReceiptStatDeleted       = "DELETED"
	ReceiptStatUndeliverable = "UNDELIV"
	ReceiptStatAccepted      = "ACCEPTD"
	ReceiptStatUnknown       = "UNKNOWN"
	ReceiptStatRejected      = "REJECTD"
)

var (
	ErrInvalidReceiptFormat = errors.New("invalid receipt format")

	receiptMatches     = regexp.MustCompile(`^id:([\w\-]+) sub:(\d+) dlvrd:(\d+) submit date:(\d+) done date:(\d+) stat:(\w+) err:(\w+)$`)
	receiptFormat      = "id:%s sub:%s dlvrd:%s submit date:%s done date:%s stat:%s err:%s text:%s"
	receiptDateLayout1 = "0601021504"
	receiptDateLayout2 = "060102150405"
)

type Receipt struct {
	Id    string    // 消息的唯一标识符
	Sub   string    // 短信总提交数量
	Dlvrd string    // 成功送达的数量
	SDate time.Time // 提交时间
	DDate time.Time // 处理完成时间
	Stat  string    // 状态
	Err   string    // 错误码
	Text  string    // 错误信息
}

func (r *Receipt) Pdu(source string, dest string) *pdu.DeliverSM {
	p := pdu.NewDeliverSM().(*pdu.DeliverSM)
	p.SourceAddr = Address(1, 1, source)
	p.DestAddr = Address(5, 0, dest)
	p.EsmClass = data.SM_SMSC_DLV_RCPT_TYPE
	p.Message = MessageInBinary([]byte(r.String()))
	return p
}

func (r *Receipt) String() string {
	return fmt.Sprintf(receiptFormat, r.Id, r.Sub, r.Dlvrd, r.submitDate(), r.doneDate(), r.Stat, r.Err, r.Text)
}

func (r *Receipt) submitDate() string {
	return r.SDate.Format(receiptDateLayout1)
}

func (r *Receipt) doneDate() string {
	return r.DDate.Format(receiptDateLayout1)
}

func ParseReceipt(s string) (*Receipt, error) {
	i := strings.Index(s, " text:")
	if i == -1 {
		i = strings.Index(s, " Text:")
		if i == -1 {
			return nil, ErrInvalidReceiptFormat
		}
	}

	match := receiptMatches.FindStringSubmatch(s[:i])
	if len(match) != 8 {
		return nil, ErrInvalidReceiptFormat
	}

	date1, err := parseReceiptDate(match[4])
	if err != nil {
		return nil, ErrInvalidReceiptFormat
	}
	date2, err := parseReceiptDate(match[5])
	if err != nil {
		return nil, ErrInvalidReceiptFormat
	}
	text := ""
	if len(s) > i+6 {
		text = s[i+6:]
	}

	return &Receipt{
		Id:    match[1],
		Sub:   match[2],
		Dlvrd: match[3],
		SDate: date1,
		DDate: date2,
		Stat:  match[6],
		Err:   match[7],
		Text:  text,
	}, nil
}

func parseReceiptDate(s string) (time.Time, error) {
	date, err := time.Parse(receiptDateLayout1, s)
	if err != nil {
		date, err = time.Parse(receiptDateLayout2, s)
		if err != nil {
			return time.Now(), ErrInvalidReceiptFormat
		}
	}
	return date, nil
}
