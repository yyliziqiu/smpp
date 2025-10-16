package smpp

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/slib/slog"
	"github.com/yyliziqiu/slib/suid"

	"github.com/yyliziqiu/smpp/util"
)

func TestMain(m *testing.M) {
	prepare()
	g := m.Run()
	finally(g)
}

func prepare() {
	_ = slog.Init(slog.Config{Path: "/private/ws/self/smpp"})
	util.SetLogger(slog.New3("smpp"))
}

func finally(code int) {
	os.Exit(code)
}

func printPdu(tag string, systemId string, p pdu.PDU) {
	if p != nil {
		bs, _ := json.MarshalIndent(p, "", "  ")
		fmt.Printf("[%s:%s:%T] %s\n\n", tag, systemId, p, string(bs))
	}
}

func submitSmPdu() *pdu.SubmitSM {
	p := pdu.NewSubmitSM().(*pdu.SubmitSM)
	p.SourceAddr = Address(5, 0, "matrix")
	p.DestAddr = Address(1, 1, "6281339900520")
	p.Message = Message("68526b7e01614899")
	p.RegisteredDelivery = 1
	return p
}

func deliverSmPdu() *pdu.DeliverSM {
	dlr := Dlr{
		Id:    suid.Get(),
		Sub:   "001",
		Dlvrd: "001",
		Sd:    time.Now(),
		Dd:    time.Now(),
		Stat:  "DELIVRD",
		Err:   "000",
		Text:  "success",
	}
	return dlr.Pdu("6281339900520", "matrix")
}

var (
	_clientConnectionConfig = ClientConnectionConfig{
		Smsc:     "127.0.0.1:10088",
		SystemId: "user1",
		Password: "user1",
		BindType: pdu.Transceiver,
	}

	_serverConnectionConfig = ServerConnectionConfig{
		Authenticate: func(conn *ServerConnection, systemId string, password string) data.CommandStatusType {
			return data.ESME_ROK
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
)

func TestClientSession(t *testing.T) {
	conf := SessionConfig{
		EnquireLink: 60 * time.Second,
		AttemptDial: 10 * time.Second,
		OnReceive: func(sess *Session, p pdu.PDU) pdu.PDU {
			printPdu("received", sess.SystemId(), p)
			if p.CanResponse() {
				return p.GetResponse()
			}
			return nil
		},
		OnRespond: func(sess *Session, resp *Response) {
			// fmt.Println("user custom data: ", values)
			printPdu("response", resp.Request.SystemId, resp.Pdu)
		},
		OnClosed: func(sess *Session, reason string, desc string) {
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	sess, err := NewSession(NewClientConnection(_clientConnectionConfig), conf)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()

	go func() {
		time.Sleep(10 * time.Second)
		// sess.Close()
	}()

	for i := 0; i < 2; i++ {
		if err = sess.Write(submitSmPdu(), "123456"); err != nil {
			t.Error(err)
		}
		time.Sleep(time.Second)
	}

	time.Sleep(time.Hour)
}

func TestServerSession(t *testing.T) {
	listen, err := net.Listen("tcp", ":10088")
	if err != nil {
		panic(err)
	}

	fmt.Println("listen: ", listen.Addr())

	for {
		conn, err := listen.Accept()
		if err != nil {
			t.Error(err)
			continue
		}
		fmt.Println("accept: ", conn.RemoteAddr())
		go accept(conn)
	}
}

func accept(conn net.Conn) {
	serv := NewServerConnection(conn, _serverConnectionConfig)

	conf := SessionConfig{
		OnReceive: func(sess *Session, p pdu.PDU) pdu.PDU {
			printPdu("received", sess.SystemId(), p)
			switch p.(type) {
			case *pdu.SubmitSM:
				p2 := p.GetResponse().(*pdu.SubmitSMResp)
				p2.MessageID = suid.Get()
				return p2
			}
			if p.CanResponse() {
				return p.GetResponse()
			}
			return nil
		},
		OnRespond: func(sess *Session, resp *Response) {
			printPdu("response", resp.Request.SystemId, resp.Pdu)
		},
		OnClosed: func(sess *Session, reason string, desc string) {
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	sess, err := NewSession(serv, conf)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// 测试写
	for i := 0; i < 3; i++ {
		_ = sess.Write(deliverSmPdu(), nil)
	}

	// 测试手动关闭
	// time.Sleep(10 * time.Second)
	// sess.Close()
}
