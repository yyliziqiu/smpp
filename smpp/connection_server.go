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

func (c *ServerConnection) SystemId() string {
	return c.systemId
}

func (c *ServerConnection) BindType() pdu.BindingType {
	return c.bindType
}

func (c *ServerConnection) SelfAddr() string {
	return c.selfAddr
}

func (c *ServerConnection) PeerAddr() string {
	return c.peerAddr
}

func (c *ServerConnection) Dial() error {
	err := c.dial()
	if err != nil {
		_ = c.conn.Close()
	}
	return err
}

func (c *ServerConnection) dial() error {
	if c.conn == nil {
		return ErrConnectionIsNil
	}

	c.selfAddr = c.conn.LocalAddr().String()
	c.peerAddr = c.conn.RemoteAddr().String()

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

	c.systemId = br.SystemID
	c.bindType = br.BindingType

	status := c.conf.Authenticate(c, br.SystemID, br.Password)

	brp := br.GetResponse().(*pdu.BindResp)
	brp.Header.CommandStatus = status
	_, err := c.Write(brp)
	if err != nil {
		return err
	}

	if status != data.ESME_ROK {
		return ErrAuthFailed
	}

	return nil
}

func (c *ServerConnection) Read() (pdu.PDU, error) {
	if c.conf.ReadTimeout > 0 {
		if err := c.conn.SetReadDeadline(time.Now().Add(c.conf.ReadTimeout)); err != nil {
			return nil, err
		}
	}
	return pdu.Parse(c.conn)
}

func (c *ServerConnection) Write(p pdu.PDU) (int, error) {
	buf := pdu.NewBuffer(make([]byte, 0, 64))
	p.Marshal(buf)

	if c.conf.WriteTimeout > 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.conf.WriteTimeout)); err != nil {
			return 0, err
		}
	}

	return c.conn.Write(buf.Bytes())
}

func (c *ServerConnection) Close() error {
	// _, _ = c.Write(pdu.NewUnbind())
	return c.conn.Close()
}

func (c *ServerConnection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}
