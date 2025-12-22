package smpp

import (
	"net"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
)

type Connection interface {
	SelfAddr() string
	PeerAddr() string
	SetDeadline(time.Time) error
	SystemId() string
	BindType() pdu.BindingType
	Dial() error
	Read() (pdu.PDU, error)
	Write(pdu.PDU) (int, error)
	Close(bool) error
}

func ConnAddrs(conn net.Conn) (string, string) {
	return conn.LocalAddr().String(), conn.RemoteAddr().String()
}

func ConnRead(conn net.Conn, timeout time.Duration) (pdu.PDU, error) {
	if timeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return nil, err
		}
	}

	return pdu.Parse(conn)
}

func ConnWrite(conn net.Conn, pd pdu.PDU, timeout time.Duration) (int, error) {
	buf := pdu.NewBuffer(make([]byte, 0, 64))
	pd.Marshal(buf)

	if timeout > 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return 0, err
		}
	}

	return conn.Write(buf.Bytes())
}

func ConnClose(conn net.Conn, bye bool) error {
	if bye {
		// 主动断开链接时，发送解绑请求
		_, _ = ConnWrite(conn, pdu.NewUnbind(), 100*time.Millisecond)
		// 防止对端响应 unbind-resp 时 reset
		time.Sleep(100 * time.Millisecond)
	}
	return conn.Close()
}
