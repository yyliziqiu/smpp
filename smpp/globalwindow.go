package smpp

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

type GlobalWindow struct {
	data map[string]*GlobalWindowNode
	heap GlobalWindowHeap
	mu   sync.Mutex
}

func NewGlobalWindow(size int) *GlobalWindow {
	return &GlobalWindow{
		data: make(map[string]*GlobalWindowNode, size),
		heap: make(GlobalWindowHeap, 0, size),
	}
}

func (t *GlobalWindow) Put(req *Request, timeout int64) {
	key := t.key(req.SessionId, req.Pdu.GetSequenceNumber())

	node := &GlobalWindowNode{
		Request:  req,
		ExpireAt: time.Now().Unix() + timeout,
	}

	t.mu.Lock()
	t.data[key] = node
	t.heap.Push(node)
	t.mu.Unlock()
}

func (t *GlobalWindow) key(sessionId string, seq int32) string {
	return fmt.Sprintf("%s:%d", sessionId, seq)
}

func (t *GlobalWindow) Take(sessionId string, seq int32) (*Request, bool) {
	var (
		key = t.key(sessionId, seq)
		req *Request
	)

	t.mu.Lock()
	node, ok := t.data[key]
	if ok {
		req = node.Request
		node.Request = nil
		delete(t.data, key)
	}
	t.mu.Unlock()

	return req, ok
}

func (t *GlobalWindow) TakeTimeout() []*Request {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.heap.Len() == 0 {
		return nil
	}

	list := make([]*Request, 0, 32)
	curr := time.Now().Unix()
	for len(t.heap) > 0 {
		node := t.heap[0]
		if node.Request != nil {
			if curr < node.ExpireAt {
				break
			}
			list = append(list, node.Request)
			delete(t.data, t.key(node.Request.SessionId, node.Request.Pdu.GetSequenceNumber()))
		}
		heap.Pop(&t.heap)
	}

	return list
}

type GlobalWindowHeap []*GlobalWindowNode

type GlobalWindowNode struct {
	Request  *Request
	ExpireAt int64
}

func (h *GlobalWindowHeap) Len() int {
	return len(*h)
}

func (h *GlobalWindowHeap) Less(i int, j int) bool {
	a := *h
	return a[i].ExpireAt < a[j].ExpireAt
}

func (h *GlobalWindowHeap) Swap(i int, j int) {
	a := *h
	a[i], a[j] = a[j], a[i]
}

func (h *GlobalWindowHeap) Push(v any) {
	*h = append(*h, v.(*GlobalWindowNode))
}

func (h *GlobalWindowHeap) Pop() any {
	a := *h
	n := len(a)
	v := a[n-1]
	a[n-1] = nil
	*h = a[0 : n-1]
	return v
}
