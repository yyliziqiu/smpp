package example

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/gdk/xuid"

	"github.com/yyliziqiu/smpp/smpp"
)

func StartServer() {
	listen, err := net.Listen("tcp", ":10032")
	if err != nil {
		panic(err)
	}

	fmt.Println("Start server on port 10032...")

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("Error: ", err)
			continue
		}
		go accept(conn)
	}
}

func accept(conn net.Conn) {
	// create server connection
	serv := smpp.NewServerConnection(conn, smpp.ServerConnectionConfig{
		// invoked when a new connection coming
		Authenticate: func(conn *smpp.ServerConnection, systemId string, password string) data.CommandStatusType {
			return data.ESME_ROK
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	// set session config
	conf := smpp.SessionConfig{
		OnReceive: func(sess *smpp.Session, p pdu.PDU) pdu.PDU {
			smpp.PrintPdu("received", sess.SystemId(), p)
			switch p.(type) {
			case *pdu.SubmitSM:
				p2 := p.GetResponse().(*pdu.SubmitSMResp)
				p2.MessageID = xuid.Get()
				return p2
			}
			if p.CanResponse() {
				return p.GetResponse()
			}
			return nil
		},
		OnRespond: func(sess *smpp.Session, resp *smpp.Response) {
			smpp.PrintPdu("response", resp.Request.SystemId, resp.Pdu)
		},
		OnClosed: func(sess *smpp.Session, reason string, desc string) {
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	// create session
	sess, err := smpp.NewSession(serv, conf)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// deliver pdu to client
	time.Sleep(3 * time.Second)
	for i := 0; i < 2; i++ {
		_ = sess.Write(deliverSmPdu(), nil)
	}
}

func deliverSmPdu() *pdu.DeliverSM {
	dlr := smpp.Dlr{
		Id:    xuid.Get(),
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
