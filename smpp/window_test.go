package smpp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/slib/sq"
	"github.com/yyliziqiu/slib/stime"
)

func TestQueueWindow(t *testing.T) {
	put := 7
	size := 5
	wait := 5 * time.Second

	w := &QueueWindow{
		size:  size,
		wait:  int64(wait.Seconds()),
		data:  make(map[int32]*QueueWindowValue, size),
		queue: sq.New(size * 2),
	}

	for i := 0; i < put; i++ {
		request := &TRequest{
			SubmitAt: time.Now().Unix(),
			Pdu:      pdu.NewSubmitSM(),
		}
		err := w.Put(request)
		if err != nil {
			fmt.Println(err)
		}
		// fmt.Printf("[put] sequence: %d\n", request.Pdu.GetSequenceNumber())
		if i%2 == 0 {
			fmt.Printf("[take] sequence: %d\n", request.Pdu.GetSequenceNumber())
			w.Take(request.Pdu.GetSequenceNumber())
		}
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			requests := w.TakeTimeout()
			for _, request := range requests {
				fmt.Printf("[timeout] sequence: %-4d, t: %d\n", request.Pdu.GetSequenceNumber(), time.Now().Unix()-request.SubmitAt)
			}
			fmt.Printf("[stat] map: %4d, queue: %s\n", len(w.data), w.queue.Status())
		}
	}()

	select {}
}

func TestQueueWindow2(t *testing.T) {
	put := 1000000
	size := 1000000
	wait := 5 * time.Second

	w := &QueueWindow{
		size:  size,
		wait:  int64(wait.Seconds()),
		data:  make(map[int32]*QueueWindowValue, size),
		queue: sq.New(size * 2),
	}

	for i := 0; i < put; i++ {
		request := &TRequest{
			SubmitAt: time.Now().Unix(),
			Pdu:      pdu.NewSubmitSM(),
		}
		err := w.Put(request)
		if err != nil {
			fmt.Println(err)
		}
	}

	printMemory("put", true)

	for k := range w.data {
		if k%2 == 0 {
			w.Take(k)
		}
	}

	printMemory("take", true)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		fmt.Printf("[stat] map: %d, queue: %s\n", len(w.data), w.queue.Status())
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				timer := stime.NewTimer()
				requests := w.TakeTimeout() // 遍历1000000耗时65ms
				if len(requests) > 0 {
					fmt.Printf("[stat] take: %d, cost: %s, map: %d, queue: %s\n", len(requests), timer.Stops(), len(w.data), w.queue.Status())
				}
			}
		}
	}()

	time.Sleep(wait + 3*time.Second)
	printMemory("clear timeout", true)

	cancel()
	time.Sleep(time.Second)
	w = nil

	printMemory("clear all", true)
}
