package smpp

import (
	"net"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

type ClientConnection struct {
	conf     *ClientConnectionConfig
	conn     net.Conn
	selfAddr string
	peerAddr string
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

func (c *ClientConnection) SelfAddr() string {
	return c.selfAddr
}

func (c *ClientConnection) PeerAddr() string {
	return c.peerAddr
}

func (c *ClientConnection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *ClientConnection) SystemId() string {
	return c.conf.SystemId
}

func (c *ClientConnection) BindType() pdu.BindingType {
	return c.conf.BindType
}

func (c *ClientConnection) Dial() error {
	// 关闭旧链接
	if c.conn != nil {
		_ = c.conn.Close()
	}

	// 连接
	var err error
	c.conn, err = c.conf.Dial(c.conf.Smsc)
	if err != nil {
		return err
	}

	// 获取两端地址
	c.selfAddr, c.peerAddr = ConnAddrs(c.conn)

	// 绑定账号
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
	return ConnRead(c.conn, c.conf.ReadTimeout)
}

func (c *ClientConnection) Write(pd pdu.PDU) (int, error) {
	return ConnWrite(c.conn, pd, c.conf.WriteTimeout)
}

func (c *ClientConnection) Close(bye bool) error {
	return ConnClose(c.conn, bye)
}
