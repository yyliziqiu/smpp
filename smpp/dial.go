package smpp

import (
	"crypto/tls"
	"net"
	"time"
)

var (
	DefaultDial = TcpDial(0)

	DefaultTlsDial = TlsDial("", 0)
)

type Dial func(addr string) (net.Conn, error)

func TcpDial(timeout time.Duration) Dial {
	return func(addr string) (net.Conn, error) {
		return net.DialTimeout("tcp", addr, timeout)
	}
}

func TlsDial(domain string, timeout time.Duration) Dial {
	return func(addr string) (net.Conn, error) {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return nil, err
		}

		cli := tls.Client(conn, &tls.Config{
			InsecureSkipVerify: domain == "",
			ServerName:         domain,
		})

		err = cli.Handshake()
		if err != nil {
			_ = conn.Close()
			return nil, err
		}

		return cli, nil
	}
}
