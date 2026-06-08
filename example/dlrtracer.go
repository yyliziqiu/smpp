package example

import (
	"container/heap"
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/yyliziqiu/smpp/libs/xuid"
)

func StartDlrTracer() {
	size := 1000
	wait := time.Minute

	// create a tracer
	t := NewDlrTracer(size)

	// put the message id to tracer for dlr trace
	t.Put(&DlrNode{
		MessageId: xuid.Get(),
		SystemId:  "user1",
		ExpireAt:  time.Now().Unix() + int64(rand.IntN(int(wait.Seconds()))),
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

type DlrTracer struct {
	data map[string]*DlrNode
	heap DlrHeap

	mu sync.Mutex
}

func NewDlrTracer(size int) *DlrTracer {
	t := &DlrTracer{
		data: make(map[string]*DlrNode, size),
		heap: make(DlrHeap, 0, size),
	}
	return t
}

func (t *DlrTracer) Put(dn *DlrNode) {
	t.mu.Lock()
	t.data[dn.MessageId] = dn
	t.heap.Push(dn)
	t.mu.Unlock()
}

func (t *DlrTracer) Take(messageId string) *DlrNode {
	t.mu.Lock()
	dn, ok := t.data[messageId]
	if ok {
		dn.took = 1
		delete(t.data, messageId)
	}
	t.mu.Unlock()

	return dn
}

func (t *DlrTracer) TakeTimeout() []*DlrNode {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.heap.Len() == 0 {
		return nil
	}

	list := make([]*DlrNode, 0, 32)
	curr := time.Now().Unix()
	for len(t.heap) > 0 {
		dn := t.heap[0]
		if dn.took == 0 {
			if curr < dn.ExpireAt {
				break
			}
			list = append(list, dn)
			dn.took = 1
			delete(t.data, dn.MessageId)
		}
		heap.Pop(&t.heap)
	}

	return list
}

type DlrHeap []*DlrNode

type DlrNode struct {
	MessageId string `json:"m"`
	SystemId  string `json:"s"`
	ExpireAt  int64  `json:"e"`
	took      int8
}

func (h *DlrHeap) Len() int {
	return len(*h)
}

func (h *DlrHeap) Less(i int, j int) bool {
	a := *h
	return a[i].ExpireAt < a[j].ExpireAt
}

func (h *DlrHeap) Swap(i int, j int) {
	a := *h
	a[i], a[j] = a[j], a[i]
}

func (h *DlrHeap) Push(v any) {
	*h = append(*h, v.(*DlrNode))
}

func (h *DlrHeap) Pop() any {
	a := *h
	n := len(a)
	v := a[n-1]
	a[n-1] = nil
	*h = a[0 : n-1]
	return v
}
