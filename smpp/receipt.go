package smpp

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"

	"github.com/yyliziqiu/smpp/libs/xconv"
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

const (
	receiptDateFormat1 = "0601021504"
	receiptDateFormat2 = "060102150405"
	receiptDateFormat3 = "200601021504"
	receiptDateFormat4 = "20060102150405"
)

var (
	ErrInvalidReceipt = errors.New("invalid receipt")
)

var (
	receiptRegexp = regexp.MustCompile(`^id:([\w\-]+) sub:(\d+) dlvrd:(\d+) submit date:(\d+) done date:(\d+) stat:(\w+) err:(\w+)$`)
	receiptFormat = "id:%s sub:%s dlvrd:%s submit date:%s done date:%s stat:%s err:%s text:%s"
)

// Receipt 消息回执 DLR
type Receipt struct {
	Id    string    // 消息 ID
	Sub   string    // 分片序号
	Dlvrd string    // 分片总数
	Sd    time.Time // 提交时间
	Dd    time.Time // 送达时间
	Stat  string    // 状态
	Err   string    // 错误码
	Text  string    // 错误描述
}

func (t *Receipt) String() string {
	return fmt.Sprintf(receiptFormat, t.Id, t.Sub, t.Dlvrd, t.Sd.Format(receiptDateFormat1), t.Dd.Format(receiptDateFormat1), t.Stat, t.Err, t.Text)
}

func (t *Receipt) DeliverSm(source string, dest string, message pdu.ShortMessage) *pdu.DeliverSM {
	p := pdu.NewDeliverSM().(*pdu.DeliverSM)
	p.SourceAddr = Address(1, 1, source)
	p.DestAddr = Address(5, 0, dest)
	p.EsmClass = data.SM_SMSC_DLV_RCPT_TYPE
	p.Message = message
	return p
}

func (t *Receipt) Pdu(source string, dest string) *pdu.DeliverSM {
	return t.DeliverSm(source, dest, BinaryMessage([]byte(t.String())))
}

func (t *Receipt) PduGsm7bit(source string, dest string) *pdu.DeliverSM {
	return t.DeliverSm(source, dest, Gsm7bitMessage(t.String()))
}

func (t *Receipt) PduUcs2(source string, dest string) *pdu.DeliverSM {
	return t.DeliverSm(source, dest, Ucs2Message(t.String()))
}

// BuildReceipt 构建回执
func BuildReceipt(id string, sub int, dlvrd int, stat string, err int) Receipt {
	curr := time.Now()
	return Receipt{
		Id:    id,
		Sub:   receiptDigit(sub),
		Dlvrd: receiptDigit(dlvrd),
		Sd:    curr,
		Dd:    curr,
		Stat:  stat,
		Err:   receiptDigit(err),
		Text:  stat,
	}
}

func receiptDigit(n int) string {
	if n < 0 || n > 999 {
		return "999"
	}
	b := make([]byte, 3)
	b[2] = byte('0' + n%10)
	n /= 10
	b[1] = byte('0' + n%10)
	n /= 10
	b[0] = byte('0' + n%10)
	return string(b)
}

// ParseReceipt 解析回执字符串
func ParseReceipt(s string) (Receipt, error) {
	i := strings.Index(s, " text:")
	if i == -1 {
		i = strings.Index(s, " Text:")
		if i == -1 {
			return Receipt{}, ErrInvalidReceipt
		}
	}

	match := receiptRegexp.FindStringSubmatch(s[:i])
	if len(match) != 8 {
		return Receipt{}, ErrInvalidReceipt
	}

	text := ""
	if len(s) > i+6 {
		text = s[i+6:]
	}

	return Receipt{
		Id:    match[1],
		Sub:   match[2],
		Dlvrd: match[3],
		Sd:    receiptDate(match[4]),
		Dd:    receiptDate(match[5]),
		Stat:  match[6],
		Err:   match[7],
		Text:  text,
	}, nil
}

func receiptDate(s string) time.Time {
	switch len(s) {
	case 10:
		if s[0:2] < "26" {
			return time.Unix(xconv.S2I64(s), 0)
		}
		if date, err := time.Parse(receiptDateFormat1, s); err == nil {
			return date
		}
	case 12:
		if date, err := time.Parse(receiptDateFormat2, s); err == nil {
			return date
		}
		if date, err := time.Parse(receiptDateFormat3, s); err == nil {
			return date
		}
	case 13:
		return time.Unix(xconv.S2I64(s)/1000, 0)
	case 14:
		if date, err := time.Parse(receiptDateFormat4, s); err == nil {
			return date
		}
	}
	return time.Now()
}
