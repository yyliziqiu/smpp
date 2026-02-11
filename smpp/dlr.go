package smpp

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/gdk/xconv"
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

	dlrRegexp      = regexp.MustCompile(`^id:([\w\-]+) sub:(\d+) dlvrd:(\d+) submit date:(\d+) done date:(\d+) stat:(\w+) err:(\w+)$`)
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

func ParseDlr(s string) (Dlr, error) {
	i := strings.Index(s, " text:")
	if i == -1 {
		i = strings.Index(s, " Text:")
		if i == -1 {
			return Dlr{}, ErrInvalidDlrFormat
		}
	}

	match := dlrRegexp.FindStringSubmatch(s[:i])
	if len(match) != 8 {
		return Dlr{}, ErrInvalidDlrFormat
	}

	text := ""
	if len(s) > i+6 {
		text = s[i+6:]
	}

	return Dlr{
		Id:    match[1],
		Sub:   match[2],
		Dlvrd: match[3],
		Sd:    parseDlrDate(match[4]),
		Dd:    parseDlrDate(match[5]),
		Stat:  match[6],
		Err:   match[7],
		Text:  text,
	}, nil
}

func parseDlrDate(s string) time.Time {
	date, err := time.Parse(dlrDateFormat1, s)
	if err == nil {
		return date
	}
	date, err = time.Parse(dlrDateFormat2, s)
	if err == nil {
		return date
	}
	if len(s) == 10 {
		return time.Unix(xconv.S2I64(s), 0)
	}
	if len(s) == 13 {
		return time.Unix(xconv.S2I64(s)/1000, 0)
	}
	return time.Now()
}

// BuildDlr
// sub   分片序号
// dlvrd 分片总数
func BuildDlr(id string, sub int, dlvrd int, stat string, err int) Dlr {
	curr := time.Now()
	return Dlr{
		Id:    id,
		Sub:   buildDlrNum(sub),
		Dlvrd: buildDlrNum(dlvrd),
		Sd:    curr,
		Dd:    curr,
		Stat:  stat,
		Err:   buildDlrNum(err),
		Text:  stat,
	}
}

func buildDlrNum(n int) string {
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
