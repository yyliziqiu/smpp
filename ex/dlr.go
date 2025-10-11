package ex

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/yyliziqiu/slib/suid"

	"github.com/yyliziqiu/smpp/assist"
)

func StartDlrTracer() {
	size := 1000
	wait := time.Minute

	// create a tracer
	t := assist.NewDlrTracer(size)

	// put the message id to tracer for dlr trace
	t.Put(&assist.DlrNode{
		MessageId: suid.Get(),
		SystemId:  "user1",
		ExpiredAt: time.Now().Unix() + int64(rand.IntN(int(wait.Seconds()))),
	})

	// take the session by message id to send dlr to client
	_ = t.Take("message id")

	// handle the timeout message when wait dlr
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
				_ = t.TakeTimeout()
			}
		}
	}()

	select {}
}
