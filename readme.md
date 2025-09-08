# Go SMPP Library

## Feature

* smpp 3.4
* smpp client
* smpp server
* support all pdu
* support both gsm7bit and ucs2 encode
* delivery receipt trace

## Based On

* [gosmpp](https://github.com/linxGnu/gosmpp)

## Install

`go get github.com/yyliziqiu/smpp`

## Example

> [more examples](https://github.com/yyliziqiu/smpp/tree/master/example)

### Client

```
package main

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

func main() {
	StartClient()
}

func StartClient() {
	// create client connection
	conn := smpp.NewClientConnection(smpp.ClientConnectionConfig{
		// connect by tls
		// Dial:     smpp.DefaultTlsDial,
		Smsc:     "127.0.0.1:10088",
		SystemId: "user1",
		Password: "user1",
		BindType: pdu.Transceiver,
	})

	// set session config
	conf := smpp.SessionConfig{
		// heartbeat interval
		EnquireLink: 60 * time.Second,
		// redial interval, session will auto redial when the tcp connection is broke if the AttemptDial > 0
		AttemptDial: 5 * time.Second,
		// when the window size is large or request timeout is small, set the WindowType = 1
		// WindowType: 1,
		// the window size
		// WindowSize: 1000,
		// timeout of request in the window
		// WindowWait: 300 * time.Second,
		// invoked when received the non-responsive pdu
		OnReceive: func(request *smpp.RRequest, _ any) pdu.PDU {
			if request.Pdu.CanResponse() {
				return request.Pdu.GetResponse()
			}
			return nil
		},
		// invoked before submit the pdu, you can get an auto-assigned message id of the submitted pdu
		OnRequest: func(request *smpp.TRequest, _ any) {
			_ = request.MessageId
		},
		// invoked when received the responsive pdu
		// or occurred error before submit
		// or wait the response of pdu timeout
		//
		// the TResponse.Pdu must be nil if the TResponse.Error is not nil
		OnRespond: func(response *smpp.TResponse, _ any) {

		},
		// invoked after the session is closed
		OnClosed: func(sess *smpp.Session, reason string, desc string, _ any) {
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
	err = sess.Write(newSubmitSm())
	if err != nil {
		log.Println("Error: ", err)
	}

	exit()
}

func newSubmitSm() *pdu.SubmitSM {
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
```

### Server

```
package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/slib/suid"

	"github.com/yyliziqiu/smpp/smpp"
)

func main() {
	StartServer()
}

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
	dlr := smpp.Dlr{
		Id:    suid.Get(),
		Sub:   "001",
		Dlvrd: "001",
		SDate: time.Now(),
		DDate: time.Now(),
		Stat:  "DELIVRD",
		Err:   "000",
		Text:  "success",
	}
	return dlr.Pdu("6281339900520", "matrix")
}
```
