package smpp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/gdk/xtime"
)

func TestGlobalWindow(t *testing.T) {
	gw := NewGlobalWindow(1000000)
	si := "123456"

	for i := 0; i < 1000000; i++ {
		gw.Put(&Request{
			Pdu:       pdu.NewSubmitSM(),
			SessionId: si,
		}, int64(rand.IntN(15)))
	}

	PrintMemory("put", true)

	for i := 0; i < 1000000; i++ {
		if i%3 == 0 {
			gw.Take(si, int32(i))
		}
	}

	PrintMemory("take", true)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		fmt.Printf("[stat] map: %d, queue: %d\n", len(gw.data), len(gw.heap))
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				timer := xtime.NewTimer()
				requests := gw.TakeTimeout() // 遍历1000000耗时65ms
				if len(requests) > 0 {
					fmt.Printf("[stat] take: %d, cost: %s, map: %d, heap: %d\n", len(requests), timer.Stops(), len(gw.data), len(gw.heap))
				}
			}
		}
	}()

	time.Sleep(18 * time.Second)
	PrintMemory("clear timeout", true)

	cancel()
	time.Sleep(time.Second)
	gw = nil

	PrintMemory("clear all", true)
}
