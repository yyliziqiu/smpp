package smpp

import (
	"net"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

type Connection interface {
	SelfAddr() string
	PeerAddr() string
	Deadline(time.Time) error
	SystemId() string
	BindType() pdu.BindingType
	Dial() error
	Read() (pdu.PDU, error)
	Write(pdu.PDU) (int, error)
	Close(bool) error
}

func ConnectionAddrs(conn net.Conn) (string, string) {
	return conn.LocalAddr().String(), conn.RemoteAddr().String()
}

func ConnectionRead(conn net.Conn, timeout time.Duration) (pdu.PDU, error) {
	if timeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return nil, err
		}
	}

	return pdu.Parse(conn)
}

func ConnectionWrite(conn net.Conn, pd pdu.PDU, timeout time.Duration) (int, error) {
	buf := pdu.NewBuffer(make([]byte, 0, 32))
	pd.Marshal(buf)

	if timeout > 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return 0, err
		}
	}

	return conn.Write(buf.Bytes())
}

func ConnectionClose(conn net.Conn, bye bool) error {
	if bye {
		// 主动断开链接时，发送解绑请求
		_, _ = ConnectionWrite(conn, pdu.NewUnbind(), 100*time.Millisecond)
		// 防止对端响应 unbind-resp 时 reset。不读取 unbind-resp，直接关闭链接
		time.Sleep(100 * time.Millisecond)
	}
	return conn.Close()
}

// ============ Client ============

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

func (c *ClientConnection) Deadline(t time.Time) error {
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
	c.selfAddr, c.peerAddr = ConnectionAddrs(c.conn)

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
	return ConnectionRead(c.conn, c.conf.ReadTimeout)
}

func (c *ClientConnection) Write(pd pdu.PDU) (int, error) {
	return ConnectionWrite(c.conn, pd, c.conf.WriteTimeout)
}

func (c *ClientConnection) Close(bye bool) error {
	return ConnectionClose(c.conn, bye)
}

// ============ Server ============

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

func (c *ServerConnection) Deadline(t time.Time) error {
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
	c.selfAddr, c.peerAddr = ConnectionAddrs(c.conn)

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
	return ConnectionRead(c.conn, c.conf.ReadTimeout)
}

func (c *ServerConnection) Write(pd pdu.PDU) (int, error) {
	return ConnectionWrite(c.conn, pd, c.conf.WriteTimeout)
}

func (c *ServerConnection) Close(bye bool) error {
	return ConnectionClose(c.conn, bye)
}
