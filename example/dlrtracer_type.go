package example

import (
	"container/heap"
	"path/filepath"
	"sync"
	"time"

	"github.com/yyliziqiu/slib/ssnap"
)

type DlrTracer struct {
	data map[string]*DlrNode
	heap DlrHeap
	snap *ssnap.Snap
	mu   sync.Mutex
}

func NewDlrTracer(size int) *DlrTracer {
	return NewDlrTracer2(size, "")
}

func NewDlrTracer2(size int, path string) *DlrTracer {
	t := &DlrTracer{
		data: make(map[string]*DlrNode, size),
		heap: make(DlrHeap, 0, size),
	}
	if path != "" {
		t.snap = ssnap.New(filepath.Join(path, "DlrTracer.data"), &t.data)
	}
	return t
}

func (t *DlrTracer) Save(_ bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.snap.Save()
}

func (t *DlrTracer) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.snap.Load()
	if err != nil {
		return err
	}

	for _, node := range t.data {
		t.heap.Push(node)
	}

	return nil
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
