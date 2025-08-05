package smpp

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/slib/slog"
	"github.com/yyliziqiu/slib/suid"
)

func TestMain(m *testing.M) {
	prepare()
	g := m.Run()
	finally(g)
}

func prepare() {
	_ = slog.Init(slog.Config{Path: "/private/ws/self/smpp"})
	SetLogger(slog.New3("smpp"))
}

func finally(code int) {
	os.Exit(code)
}

var (
	_clientConnectionConfig = ClientConnectionConfig{
		Smsc:     "127.0.0.1:10088",
		SystemId: "user1",
		Password: "user1",
		BindType: pdu.Transceiver,
	}

	_serverConnectionConfig = ServerConnectionConfig{
		Authenticate: func(systemId string, password string) bool {
			return systemId == "user1" && password == "user1"
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
)

func TestClientSession(t *testing.T) {
	conf := SessionConfig{
		EnquireLink: 60 * time.Second,
		AttemptDial: 10 * time.Second,
		Values:      "this is a test session",
		OnReceive: func(request *RRequest, _ any) pdu.PDU {
			logTest("received", request.Session.SystemId(), request.Pdu)
			if request.Pdu.CanResponse() {
				return request.Pdu.GetResponse()
			}
			return nil
		},
		OnRespond: func(response *TResponse, values any) {
			// fmt.Println("user custom data: ", values)
			logTest("response", response.Request.SystemId, response.Pdu)
		},
		OnClosed: func(sess *Session, reason string, desc string, _ any) {
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
		if err = sess.Write(newTestSubmitSM()); err != nil {
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
	connect := NewServerConnection(conn, _serverConnectionConfig)

	conf := SessionConfig{
		OnReceive: func(request *RRequest, _ any) pdu.PDU {
			logTest("received", request.Session.SystemId(), request.Pdu)
			switch request.Pdu.(type) {
			case *pdu.SubmitSM:
				p := request.Pdu.GetResponse().(*pdu.SubmitSMResp)
				p.MessageID = suid.Get()
				return p
			}
			if request.Pdu.CanResponse() {
				return request.Pdu.GetResponse()
			}
			return nil
		},
		OnRespond: func(response *TResponse, _ any) {
			logTest("response", response.Request.SystemId, response.Pdu)
		},
		OnClosed: func(sess *Session, reason string, desc string, _ any) {
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	sess, err := NewSession(connect, conf)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// 测试写
	for i := 0; i < 3; i++ {
		_ = sess.Write(newTestDeliverSM())
	}

	// time.Sleep(10 * time.Second)
	// sess.Close() // 测试手动关闭
}
