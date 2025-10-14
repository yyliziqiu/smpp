package assist

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/yyliziqiu/slib/slog"
	"github.com/yyliziqiu/slib/stime"
	"github.com/yyliziqiu/slib/suid"

	"github.com/yyliziqiu/smpp/util"
)

func TestMain(m *testing.M) {
	prepare()
	g := m.Run()
	finally(g)
}

func prepare() {
	_ = slog.Init(slog.Config{Path: "/private/ws/self/smpp"})
	util.SetLogger(slog.New3("assist"))
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
			MessageId: suid.Get(),
			SystemId:  "user1",
			ExpireAt:  time.Now().Unix() + int64(rand.IntN(int(wait.Seconds()))),
		})
	}

	util.PrintMemory("put", true)

	ti := 0
	for k := range w.data {
		if ti < put/2 {
			w.Take(k)
			ti++
		}
	}

	util.PrintMemory("take", true)

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
				timer := stime.NewTimer()
				tos := w.TakeTimeout() // 遍历1000000耗时150ms
				if len(tos) > 0 {
					fmt.Printf("[stat] take: %d, cost: %s, map: %d, heap: %d\n", len(tos), timer.Stops(), len(w.data), w.heap.Len())
				}
			}
		}
	}()

	time.Sleep(wait + 3*time.Second)
	util.PrintMemory("clear timeout", true)

	cancel()
	time.Sleep(time.Second)
	w = nil

	util.PrintMemory("clear all", true)
}

func TestNewDlrTracer2(t *testing.T) {
	w := NewDlrTracer2(10, "/private/ws/self/smpp/data")

	for i := 0; i < 3; i++ {
		w.Put(&DlrNode{
			MessageId: suid.Get(),
			SystemId:  "user1",
			ExpireAt:  time.Now().Unix() + int64(i),
		})
	}

	err := w.Save(false)
	if err != nil {
		t.Fatal(err)
	}

	w = NewDlrTracer2(10, "/private/ws/self/smpp/data")

	err = w.Load()
	if err != nil {
		t.Fatal(err)
	}

	bs, _ := json.MarshalIndent(w.data, "", "  ")
	fmt.Println(string(bs))
	fmt.Println("*****************************")
	bs, _ = json.MarshalIndent(w.heap, "", "  ")
	fmt.Println(string(bs))
}
