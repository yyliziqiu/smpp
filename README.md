# smpp

A clean and production-ready **SMPP 3.4** library for Go, built on top of [gosmpp](https://github.com/linxGnu/gosmpp).

[![Go Version](https://img.shields.io/badge/go-1.24%2B-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

---

## Features

- **SMPP 3.4** protocol — full PDU support
- **Client & Server** — both sides in a unified session model
- **Auto-reconnect** — configurable redial interval, transparent to callers
- **Flow-control window** — two implementations tuned for different throughput profiles
- **Delivery receipts** — parse, build, and encode DLR payloads
- **Message helpers** — auto-detect encoding, GSM-7bit, UCS-2, Binary
- **Heartbeat** — built-in EnquireLink loop
- **TLS** — optional TLS dial

---

## Installation

```bash
go get github.com/yyliziqiu/smpp
```

---

## Quick Start

### Client

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/smpp/smpp"
)

func main() {
	conn := smpp.NewClientConnection(smpp.ClientConnectionConfig{
		Smsc:     "127.0.0.1:10032",
		SystemId: "user1",
		Password: "user1",
		BindType: pdu.Transceiver,
	})

	sess, err := smpp.NewSession(conn, smpp.SessionConfig{
		EnquireLink: 15 * time.Second,
		AttemptDial: 5 * time.Second,
		OnReceive: func(sess *smpp.Session, p pdu.PDU) pdu.PDU {
			if p.CanResponse() {
				return p.GetResponse()
			}
			return nil
		},
		OnRespond: func(sess *smpp.Session, resp *smpp.Response) {
			if resp.Error != nil {
				fmt.Println("response error:", resp.Error)
			}
		},
	})
	if err != nil {
		panic(err)
	}
	defer sess.Close()

	p := pdu.NewSubmitSM().(*pdu.SubmitSM)
	p.SourceAddr = smpp.Address(5, 0, "MyBrand")
	p.DestAddr = smpp.Address(1, 1, "8613800000000")
	p.Message = smpp.Message("Hello, world!")
	p.RegisteredDelivery = 1

	if err := sess.Write(p, nil); err != nil {
		fmt.Println("write error:", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

```

### Server

```go
package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/smpp/smpp"
)

func main() {
	listen, err := net.Listen("tcp", ":10032")
	if err != nil {
		panic(err)
	}
	fmt.Println("listening on :10032")

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	serv := smpp.NewServerConnection(conn, smpp.ServerConnectionConfig{
		Authenticate: func(_ *smpp.ServerConnection, _, _ string) data.CommandStatusType {
			return data.ESME_ROK
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Second,
	})
	sess, err := smpp.NewSession(serv, smpp.SessionConfig{
		OnReceive: func(sess *smpp.Session, p pdu.PDU) pdu.PDU {
			if p.CanResponse() {
				return p.GetResponse()
			}
			return nil
		},
		OnClosed: func(sess *smpp.Session, reason, desc string) {
			fmt.Printf("closed: system_id=%s reason=%s\n", sess.SystemId(), reason)
		},
	})
	if err != nil {
		return
	}
	_ = sess
}
```

---

## Session Config

| Field         | Type                                  | Description                                                                  |
|---------------|---------------------------------------|------------------------------------------------------------------------------|
| `Context`     | `any`                                 | Arbitrary user data attached to the session                                  |
| `EnquireLink` | `time.Duration`                       | Heartbeat interval (0 = disabled)                                            |
| `AttemptDial` | `time.Duration`                       | Redial interval on disconnect (0 = no reconnect)                             |
| `WindowType`  | `int`                                 | `0` SmallWindow (default), `1` LargeWindow                                   |
| `WindowSize`  | `int`                                 | Max in-flight requests (default 32)                                          |
| `WindowWait`  | `time.Duration`                       | Request timeout (default 10s)                                                |
| `WindowScan`  | `time.Duration`                       | Interval to sweep timed-out requests (default 30s)                           |
| `WindowBlock` | `time.Duration`                       | Block behavior when window is full: `0` return error, `>0` sleep, `<0` yield |
| `OnDialed`    | `func(*Session)`                      | Called after each successful (re)connect                                     |
| `OnClosed`    | `func(*Session, reason, desc string)` | Called when the session is fully closed                                      |
| `OnReceive`   | `func(*Session, PDU) PDU`             | Called for every inbound non-response PDU; return a PDU to reply             |
| `OnRequest`   | `func(*Session, *Request)`            | Called before each user-submitted PDU is sent                                |
| `OnRespond`   | `func(*Session, *Response)`           | Called when a response arrives, times out, or errors                         |

---

## Window

The window controls how many requests can be in-flight at the same time.

| Implementation          | Best for                                            |
|-------------------------|-----------------------------------------------------|
| `SmallWindow` (default) | Low concurrency, small window sizes                 |
| `LargeWindow`           | High throughput, large window sizes, short timeouts |

Switch via `SessionConfig.WindowType = 1` or provide a custom factory through `WindowNewer`.

---

## Delivery Receipts (DLR)

```go
package main

import (
	"fmt"

	"github.com/yyliziqiu/smpp/smpp"
)

func main() {
	// Build a DLR
	dlr := smpp.BuildDlr("msg-001", 1, 1, smpp.DlrStatDelivered, 0)
	fmt.Println(dlr.String())

	// Parse a DLR string
	parsed, err := smpp.ParseDlr(dlr.String())
	if err != nil {
		panic(err)
	}
	fmt.Printf("id=%s stat=%s\n", parsed.Id, parsed.Stat)

	// Encode into a DeliverSM PDU (to write over a session)
	_ = dlr.Pdu("6281339900520", "MyBrand")        // binary (default)
	_ = dlr.PduGsm7bit("6281339900520", "MyBrand") // GSM-7bit
	_ = dlr.PduUcs2("6281339900520", "MyBrand")    // UCS-2
}
```

DLR status constants: `DlrStatDelivered`, `DlrStatUndeliverable`, `DlrStatExpired`, `DlrStatRejected`, and more.

---

## Message Helpers

```go
package main

import (
	"fmt"

	"github.com/yyliziqiu/smpp/smpp"
)

func main() {
	_ = smpp.Message("hello")            // auto-detect GSM-7bit or UCS-2
	_ = smpp.Gsm7bitMessage("hello")     // force GSM-7bit
	_ = smpp.Ucs2Message("你好")         // force UCS-2
	_ = smpp.BinaryMessage([]byte{0x01}) // binary (8-bit)

	msgLen, segments, isGsm := smpp.DetectMessage("Hello, world!")
	fmt.Printf("len=%d segments=%d gsm=%v\n", msgLen, segments, isGsm)
}
```

---

## TLS

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/smpp/smpp"
)

func main() {
	conn := smpp.NewClientConnection(smpp.ClientConnectionConfig{
		Dial:     smpp.DefaultTlsDial,
		Smsc:     "smsc.example.com:10035",
		SystemId: "user1",
		Password: "user1",
		BindType: pdu.Transceiver,
	})

	sess, err := smpp.NewSession(conn, smpp.SessionConfig{
		EnquireLink: 15 * time.Second,
		OnReceive: func(sess *smpp.Session, p pdu.PDU) pdu.PDU {
			if p.CanResponse() {
				return p.GetResponse()
			}
			return nil
		},
	})
	if err != nil {
		panic(err)
	}
	defer sess.Close()

	fmt.Println("connected via TLS, session id:", sess.Id())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
```

---

## Global Session Store

Every active session is automatically registered in a global store.

```go
package main

import (
	"fmt"

	"github.com/yyliziqiu/smpp/smpp"
)

func main() {
	// These are available after sessions are created elsewhere in your program.
	sess := smpp.GetSession("session-id")
	fmt.Println(sess)

	all := smpp.GetSessions()
	fmt.Println("active sessions:", len(all))

	fmt.Println("session count:", smpp.CountSessions())
}
```

---

## Examples

Full runnable examples live in the [`example/`](example/) directory.

```bash
# run the server
go run . server

# run the client (in another terminal)
go run . client
```

---

## Based On

- [gosmpp](https://github.com/linxGnu/gosmpp) — low-level SMPP PDU codec
