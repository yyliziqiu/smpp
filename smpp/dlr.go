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
	DlrStatEnRoute       = "ENROUTE"
	DlrStatDelivered     = "DELIVRD"
	DlrStatExpired       = "EXPIRED"
	DlrStatDeleted       = "DELETED"
	DlrStatUndeliverable = "UNDELIV"
	DlrStatAccepted      = "ACCEPTD"
	DlrStatUnknown       = "UNKNOWN"
	DlrStatRejected      = "REJECTD"
)

var (
	ErrInvalidDlrFormat = errors.New("invalid dlr format")

	dlrMatches     = regexp.MustCompile(`^id:([\w\-]+) sub:(\d+) dlvrd:(\d+) submit date:(\d+) done date:(\d+) stat:(\w+) err:(\w+)$`)
	dlrFormat      = "id:%s sub:%s dlvrd:%s submit date:%s done date:%s stat:%s err:%s text:%s"
	dlrDateFormat1 = "0601021504"
	dlrDateFormat2 = "060102150405"
)

type Dlr struct {
	Id    string    // 消息的唯一标识符
	Sub   string    // 短信总提交数量
	Dlvrd string    // 成功送达的数量
	Sd    time.Time // 提交时间
	Dd    time.Time // 处理完成时间
	Stat  string    // 状态
	Err   string    // 错误码
	Text  string    // 错误信息
}

func (r *Dlr) String() string {
	return fmt.Sprintf(dlrFormat, r.Id, r.Sub, r.Dlvrd, r.Sd.Format(dlrDateFormat1), r.Dd.Format(dlrDateFormat1), r.Stat, r.Err, r.Text)
}

func (r *Dlr) Pdu(source string, dest string) *pdu.DeliverSM {
	p := pdu.NewDeliverSM().(*pdu.DeliverSM)
	p.SourceAddr = Address(1, 1, source)
	p.DestAddr = Address(5, 0, dest)
	p.EsmClass = data.SM_SMSC_DLV_RCPT_TYPE
	p.Message = BinaryMessage([]byte(r.String()))
	return p
}

func ParseDlr(s string) (*Dlr, error) {
	i := strings.Index(s, " text:")
	if i == -1 {
		i = strings.Index(s, " Text:")
		if i == -1 {
			return nil, ErrInvalidDlrFormat
		}
	}

	match := dlrMatches.FindStringSubmatch(s[:i])
	if len(match) != 8 {
		return nil, ErrInvalidDlrFormat
	}

	date1, err := parseDlrDate(match[4])
	if err != nil {
		return nil, ErrInvalidDlrFormat
	}
	date2, err := parseDlrDate(match[5])
	if err != nil {
		return nil, ErrInvalidDlrFormat
	}
	text := ""
	if len(s) > i+6 {
		text = s[i+6:]
	}

	return &Dlr{
		Id:    match[1],
		Sub:   match[2],
		Dlvrd: match[3],
		Sd:    date1,
		Dd:    date2,
		Stat:  match[6],
		Err:   match[7],
		Text:  text,
	}, nil
}

func parseDlrDate(s string) (time.Time, error) {
	date, err := time.Parse(dlrDateFormat1, s)
	if err != nil {
		date, err = time.Parse(dlrDateFormat2, s)
		if err != nil {
			return time.Now(), ErrInvalidDlrFormat
		}
	}
	return date, nil
}
