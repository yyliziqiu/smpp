package smpp

import (
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

func TestClientSession(t *testing.T) {
	cc := ClientConnectionConfig{
		Smsc:     "127.0.0.1:10032",
		SystemId: "test_user",
		Password: "test_user",
		BindType: pdu.Transceiver,
	}

	sc := SessionConfig{
		EnquireLink: 10 * time.Second,
		AttemptDial: 10 * time.Second,
		OnReceive: func(sess *Session, p pdu.PDU) pdu.PDU {
			util.PrintPdu("received", sess.SystemId(), p)
			if p.CanResponse() {
				return p.GetResponse()
			}
			return nil
		},
		OnRespond: func(sess *Session, resp *Response) {
			// fmt.Println("user custom data: ", values)
			util.PrintPdu("response", resp.Request.SystemId, resp.Pdu)
		},
		OnClosed: func(sess *Session, reason string, desc string) {
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	sess, err := NewSession(NewClientConnection(cc), sc)
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

func submitSmPdu() *pdu.SubmitSM {
	p := pdu.NewSubmitSM().(*pdu.SubmitSM)
	p.SourceAddr = Address(5, 0, "matrix")
	p.DestAddr = Address(1, 1, "6281339900520")
	p.Message = Message("68526b7e01614899")
	p.RegisteredDelivery = 1
	return p
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
	cc := ServerConnectionConfig{
		Authenticate: func(conn *ServerConnection, systemId string, password string) data.CommandStatusType {
			return data.ESME_ROK
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	sc := SessionConfig{
		OnReceive: func(sess *Session, p pdu.PDU) pdu.PDU {
			util.PrintPdu("received", sess.SystemId(), p)
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
			util.PrintPdu("response", resp.Request.SystemId, resp.Pdu)
		},
		OnClosed: func(sess *Session, reason string, desc string) {
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	sess, err := NewSession(NewServerConnection(conn, cc), sc)
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
