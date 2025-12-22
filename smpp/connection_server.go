package smpp

import (
	"net"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

type ServerConnection struct {
	conf     *ServerConnectionConfig
	conn     net.Conn
	systemId string
	bindType pdu.BindingType
	selfAddr string
	peerAddr string
}

type ServerConnectionConfig struct {
	Authenticate ServerConnectionAuthenticate
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type ServerConnectionAuthenticate func(conn *ServerConnection, systemId string, password string) data.CommandStatusType

func NewServerConnection(conn net.Conn, conf ServerConnectionConfig) *ServerConnection {
	return &ServerConnection{conn: conn, conf: &conf}
}

func (c *ServerConnection) SelfAddr() string {
	return c.selfAddr
}

func (c *ServerConnection) PeerAddr() string {
	return c.peerAddr
}

func (c *ServerConnection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *ServerConnection) SystemId() string {
	return c.systemId
}

func (c *ServerConnection) BindType() pdu.BindingType {
	return c.bindType
}

func (c *ServerConnection) Dial() error {
	err := c.dial()
	if err != nil {
		_ = c.conn.Close()
	}
	return err
}

func (c *ServerConnection) dial() error {
	// 关闭旧链接
	if c.conn == nil {
		return ErrConnectionIsNil
	}

	// 获取两端地址
	c.selfAddr, c.peerAddr = ConnAddrs(c.conn)

	// 获取绑定请求
	var (
		br *pdu.BindRequest
		ok bool
	)
	for i := 0; i < 3; i++ {
		p, err := c.Read()
		if err != nil {
			return err
		}
		br, ok = p.(*pdu.BindRequest)
		if ok {
			break
		}
	}
	if !ok {
		return ErrBindFailed
	}

	// 记录绑定信息
	c.systemId = br.SystemID
	c.bindType = br.BindingType

	// 账户认证
	status := c.conf.Authenticate(c, br.SystemID, br.Password)

	// 返回绑定结果
	brp := br.GetResponse().(*pdu.BindResp)
	brp.Header.CommandStatus = status
	if _, err := c.Write(brp); err != nil {
		return err
	}

	// 认证失败返回错误
	if status != data.ESME_ROK {
		return ErrAuthFailed
	}

	return nil
}

func (c *ServerConnection) Read() (pdu.PDU, error) {
	return ConnRead(c.conn, c.conf.ReadTimeout)
}

func (c *ServerConnection) Write(pd pdu.PDU) (int, error) {
	return ConnWrite(c.conn, pd, c.conf.WriteTimeout)
}

func (c *ServerConnection) Close(bye bool) error {
	return ConnClose(c.conn, bye)
}
