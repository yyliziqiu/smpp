package smpp

import (
	"net"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

type ClientConnection struct {
	conf *ClientConnectionConfig
	conn net.Conn
}

type ClientConnectionConfig struct {
	Dial         Dial
	Smsc         string
	SystemId     string
	Password     string
	BindType     pdu.BindingType
	SystemType   string
	AddressRange pdu.AddressRange
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func NewClientConnection(conf ClientConnectionConfig) *ClientConnection {
	if conf.Dial == nil {
		conf.Dial = DefaultDial
	}
	return &ClientConnection{conf: &conf}
}

func (c *ClientConnection) SystemId() string {
	return c.conf.SystemId
}

func (c *ClientConnection) BindType() pdu.BindingType {
	return c.conf.BindType
}

func (c *ClientConnection) SelfAddr() string {
	if c.conn == nil {
		return ""
	}
	return c.conn.LocalAddr().String()
}

func (c *ClientConnection) PeerAddr() string {
	if c.conn == nil {
		return c.conf.Smsc
	}
	return c.conn.RemoteAddr().String()
}

func (c *ClientConnection) Dial() error {
	if c.conn != nil {
		_ = c.conn.Close()
	}

	var err error

	c.conn, err = c.conf.Dial(c.conf.Smsc)
	if err != nil {
		return err
	}

	err = c.bind()
	if err != nil {
		_ = c.conn.Close()
		return err
	}

	return nil
}

func (c *ClientConnection) bind() error {
	// 创建绑定请求
	bp := pdu.NewBindRequest(c.conf.BindType)
	bp.SystemID = c.conf.SystemId
	bp.Password = c.conf.Password
	bp.SystemType = c.conf.SystemType
	bp.AddressRange = c.conf.AddressRange

	// 发送绑定请求
	_, err := c.Write(bp)
	if err != nil {
		return err
	}

	// 读取响应
	var (
		p  pdu.PDU
		rp *pdu.BindResp
		ok bool
	)
	for i := 0; i < 3; i++ {
		p, err = c.Read()
		if err != nil {
			return err
		}
		rp, ok = p.(*pdu.BindResp)
		if ok {
			break
		}
	}

	// 响应失败
	if !ok || bp.GetSequenceNumber() != rp.GetSequenceNumber() {
		return ErrBindFailed
	}

	// 绑定失败
	if rp.CommandStatus != data.ESME_ROK {
		err = NewStatusError(rp.CommandStatus)
		return err
	}

	return nil
}

func (c *ClientConnection) Read() (pdu.PDU, error) {
	if c.conf.ReadTimeout > 0 {
		if err := c.conn.SetReadDeadline(time.Now().Add(c.conf.ReadTimeout)); err != nil {
			return nil, err
		}
	}
	return pdu.Parse(c.conn)
}

func (c *ClientConnection) Write(p pdu.PDU) (int, error) {
	buf := pdu.NewBuffer(make([]byte, 0, 64))
	p.Marshal(buf)

	if c.conf.WriteTimeout > 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.conf.WriteTimeout)); err != nil {
			return 0, err
		}
	}

	return c.conn.Write(buf.Bytes())
}

func (c *ClientConnection) Close() error {
	_, _ = c.Write(pdu.NewUnbind())
	time.Sleep(100 * time.Millisecond) // 防止对端响应时 reset
	return c.conn.Close()
}

func (c *ClientConnection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}
