package example

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/yyliziqiu/smpp/libs/xtimer"
	"github.com/yyliziqiu/smpp/libs/xuid"
	"github.com/yyliziqiu/smpp/smpp"
)

func TestMain(m *testing.M) {
	prepare()
	g := m.Run()
	finally(g)
}

func prepare() {
	smpp.SetLog(logrus.New())
}

func finally(code int) {
	os.Exit(code)
}

func TestDlrTracer(t *testing.T) {
	put := 1000000
	size := 1000000
	wait := 20 * time.Second

	w := &DlrTracer{
		data: make(map[string]*DlrNode, size),
		heap: make(DlrHeap, 0, size),
	}

	for i := 0; i < put; i++ {
		w.Put(&DlrNode{
			MessageId: xuid.Get(),
			SystemId:  "user1",
			ExpireAt:  time.Now().Unix() + int64(rand.IntN(int(wait.Seconds()))),
		})
	}

	smpp.PrintMemory("put", true)

	ti := 0
	for k := range w.data {
		if ti < put/2 {
			w.Take(k)
			ti++
		}
	}

	smpp.PrintMemory("take", true)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		fmt.Printf("[stat] map: %d, heap: %d\n", len(w.data), w.heap.Len())
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				timer := xtimer.New()
				tos := w.TakeTimeout() // 遍历1000000耗时150ms
				if len(tos) > 0 {
					fmt.Printf("[stat] take: %d, cost: %s, map: %d, heap: %d\n", len(tos), timer.Stop(), len(w.data), w.heap.Len())
				}
			}
		}
	}()

	time.Sleep(wait + 3*time.Second)
	smpp.PrintMemory("clear timeout", true)

	cancel()
	time.Sleep(time.Second)
	w = nil

	smpp.PrintMemory("clear all", true)
}
