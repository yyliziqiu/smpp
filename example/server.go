package example

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/slib/suid"

	"github.com/yyliziqiu/smpp/smpp"
)

func StartServer() {
	listen, err := net.Listen("tcp", ":10088")
	if err != nil {
		panic(err)
	}

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
	connect := smpp.NewServerConnection(conn, smpp.ServerConnectionConfig{
		// invoked when a new connection coming
		Authenticate: func(systemId string, password string) bool {
			return systemId == "user1" && password == "user1"
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	// set session config
	conf := smpp.SessionConfig{
		OnReceive: func(request *smpp.RRequest, _ any) pdu.PDU {
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
		OnRespond: func(response *smpp.TResponse, _ any) {

		},
		OnClosed: func(sess *smpp.Session, reason string, desc string, _ any) {

		},
	}

	// create session
	sess, err := smpp.NewSession(connect, conf)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// deliver pdu to client
	_ = sess.Write(newDeliverSm())
}

func newDeliverSm() *pdu.DeliverSM {
	receipt := smpp.Receipt{
		Id:    suid.Get(),
		Sub:   "001",
		Dlvrd: "001",
		SDate: time.Now(),
		DDate: time.Now(),
		Stat:  "DELIVRD",
		Err:   "000",
		Text:  "success",
	}
	return receipt.Pdu("6281339900520", "matrix")
}
