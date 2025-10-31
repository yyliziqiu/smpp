package example

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linxGnu/gosmpp/pdu"

	"github.com/yyliziqiu/smpp/smpp"
)

func StartClient() {
	// create client connection
	conn := smpp.NewClientConnection(smpp.ClientConnectionConfig{
		// Dial:     smpp.DefaultTlsDial, // connect by tls
		Smsc:     "127.0.0.1:10088",
		SystemId: "user1",
		Password: "user1",
		BindType: pdu.Transceiver,
	})

	// set session config
	conf := smpp.SessionConfig{
		// custom user data
		Context: 1,
		// heartbeat interval
		EnquireLink: 15 * time.Second,
		// redial interval, session will auto redial when the tcp connection is broke if the AttemptDial > 0
		AttemptDial: 5 * time.Second,
		// when the window size is large or request timeout is small, set the WindowType = 1
		// WindowType: 1,
		// the window size
		// WindowSize: 1000,
		// timeout of request in the window
		// WindowWait: 300 * time.Second,
		// invoked when received the non-responsive pdu
		OnReceive: func(sess *smpp.Session, p pdu.PDU) pdu.PDU {
			smpp.PrintPdu("received", sess.SystemId(), p)
			if p.CanResponse() {
				return p.GetResponse()
			}
			return nil
		},
		// invoked when received the responsive pdu
		// or occurred error before submit
		// or wait the response of pdu timeout
		//
		// the Response.Pdu must be nil if the TResponse.Error is not nil
		OnRespond: func(sess *smpp.Session, resp *smpp.Response) {
			smpp.PrintPdu("response", resp.Request.SystemId, resp.Pdu)
		},
		// invoked after the session is closed
		OnClosed: func(sess *smpp.Session, reason string, desc string) {
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	// create session
	sess, err := smpp.NewSession(conn, conf)
	if err != nil {
		panic(err)
	}
	defer sess.Close()

	// submit pdu by session
	err = sess.Write(submitSmPdu(), "123456")
	if err != nil {
		log.Println("Error: ", err)
	}

	exit()
}

func submitSmPdu() *pdu.SubmitSM {
	p := pdu.NewSubmitSM().(*pdu.SubmitSM)
	p.SourceAddr = smpp.Address(5, 0, "matrix")
	p.DestAddr = smpp.Address(1, 1, "86387490")
	p.Message = smpp.Message("68526b7e0161489")
	p.RegisteredDelivery = 1
	return p
}

func exit() {
	exitCh := make(chan os.Signal)
	signal.Notify(exitCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-exitCh
}
