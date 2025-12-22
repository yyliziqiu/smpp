package smpp

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
	"github.com/sirupsen/logrus"
)

// ============ Logger ============

var _logger *logrus.Logger

func SetLogger(logger *logrus.Logger) {
	_logger = logger
}

// ============ Session ============

var _store = NewSessionStore()

func GetSession(id string) *Session {
	return _store.GetSession(id)
}

func GetSessions() map[string]*Session {
	return _store.GetSessions()
}

func CountSessions() int {
	return _store.CountSessions()
}

// ============ Message ============

// Address
// TON (Type of Number)
// 0  Unknown           未知类型（默认）
// 1  International     国际号码（带国家代码，如 +8613800000000）
// 2  National          国内号码（不带国家码，如 13800000000）
// 3  Network Specific  特定网络号码（内部路由号码）
// 4  Subscriber Number 用户号码（短号或本地号码）
// 5  Alphanumeric      字母数字组合（用于 Sender ID，例如 "MyBrand"）
// 6  Abbreviated       缩写号码（例如短号 12345）
//
// NPI (Numbering Plan Indicator)
// 0  Unknown            未知编号计划（默认）
// 1  ISDN / E.164       国际标准电话编号（最常见）
// 3  Data(X.121)        数据网络编号
// 4  Telex              电传编号
// 6  Land Mobile(E.212) 移动通信编号
// 8  National           国家编号计划
// 9  Private            私有编号计划
// 10 ERMES              欧洲寻呼系统编号
//
// 常用组合
// 1  1  国际号码（E.164 格式），发送到国际手机号码 +8613800000000
// 2  1  国内号码（E.164），发送到本地号码 13800000000
// 5  0  字母数字型发件人，使用品牌名作为发件人 MyBrand
// 6  0  缩写短号，使用短号发件人 12345
func Address(ton byte, npi byte, addr string) pdu.Address {
	ret := pdu.NewAddress()
	ret.SetTon(ton)
	ret.SetNpi(npi)
	_ = ret.SetAddress(addr)
	return ret
}

func Message(s string) pdu.ShortMessage {
	sm, _ := pdu.NewShortMessageWithEncoding(s, data.FindEncoding(s))
	return sm
}

func Gsm7bitMessage(s string) pdu.ShortMessage {
	sm, _ := pdu.NewShortMessageWithEncoding(s, data.GSM7BIT)
	return sm
}

func Ucs2Message(s string) pdu.ShortMessage {
	sm, _ := pdu.NewShortMessageWithEncoding(s, data.UCS2)
	return sm
}

func BinaryMessage(s []byte) pdu.ShortMessage {
	sm, _ := pdu.NewBinaryShortMessageWithEncoding(s, data.BINARY8BIT2)
	return sm
}

// ============ Debug ============

func PrintPdu(tag string, systemId string, p pdu.PDU) {
	if p != nil {
		bs, _ := json.MarshalIndent(p, "", "  ")
		fmt.Printf("[%s:%s:%T] %s\n\n", tag, systemId, p, string(bs))
	}
}

func PrintMemory(tag string, gc bool) {
	if gc {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Printf("[memory:%s] alloc: %d KB\n", tag, memStats.Alloc/1024)
}
