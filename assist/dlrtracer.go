package assist

import (
	"container/heap"
	"path/filepath"
	"sync"
	"time"

	"github.com/yyliziqiu/slib/ssnap"
)

type DlrHeap []*DlrNode

type DlrNode struct {
	MessageId string
	SystemId  string
	SessionId string
	ExpiredAt int64
	hasTook   bool
}

func (h *DlrHeap) Len() int {
	return len(*h)
}

func (h *DlrHeap) Less(i int, j int) bool {
	a := *h
	return a[i].ExpiredAt < a[j].ExpiredAt
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

type DlrTracer struct {
	data  map[string]*DlrNode
	heap  DlrHeap
	dSnap *ssnap.Snap
	hSnap *ssnap.Snap
	mu    sync.Mutex
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
		t.dSnap = ssnap.New(filepath.Join(path, "DlrTracer.data"), &t.data)
		t.hSnap = ssnap.New(filepath.Join(path, "DlrTracer.heap"), &t.heap)
	}
	return t
}

func (t *DlrTracer) Save(_ bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.dSnap.Save(); err != nil {
		return err
	}

	if err := t.hSnap.Save(); err != nil {
		return err
	}

	return nil
}

func (t *DlrTracer) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.dSnap.Load(); err != nil {
		return err
	}

	if err := t.hSnap.Load(); err != nil {
		return err
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
		dn.hasTook = true
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
		if !dn.hasTook {
			if curr < dn.ExpiredAt {
				break
			}
			list = append(list, dn)
			dn.hasTook = true
			delete(t.data, dn.MessageId)
		}
		heap.Pop(&t.heap)
	}

	return list
}
