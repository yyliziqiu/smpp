package example

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/yyliziqiu/slib/suid"

	"github.com/yyliziqiu/smpp/smpp"
)

func StartReceiptTracer() {
	size := 1000
	wait := time.Minute

	// create a tracer
	w := smpp.NewReceiptTracer(size)

	// put the message id to tracer for receipt trace
	w.Put(&smpp.ReceiptTo{
		MessageId: suid.Get(),
		SystemId:  "user1",
		ExpiredAt: time.Now().Unix() + int64(rand.IntN(int(wait.Seconds()))),
	})

	// take the session by message id to send receipt to client
	_ = w.Take("message id")

	// handle the timeout message when wait receipt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = w.TakeTimeout()
			}
		}
	}()

	select {}
}
