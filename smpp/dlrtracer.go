package smpp

import (
	"container/heap"
	"path/filepath"
	"sync"
	"time"

	"github.com/yyliziqiu/slib/ssnap"
)

type DlrEntry struct {
	MessageId string
	SystemId  string
	ExpiredAt int64
	hasTook   bool
}

type DlrHeap []*DlrEntry

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
	*h = append(*h, v.(*DlrEntry))
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
	data     map[string]*DlrEntry
	heap     DlrHeap
	dataSnap *ssnap.Snap
	heapSnap *ssnap.Snap
	mu       sync.Mutex
}

func NewDlrTracer(size int) *DlrTracer {
	return NewDlrTracer2(size, "")
}

func NewDlrTracer2(size int, path string) *DlrTracer {
	t := &DlrTracer{
		data: make(map[string]*DlrEntry, size),
		heap: make(DlrHeap, 0, size),
	}
	if path != "" {
		t.dataSnap = ssnap.New(filepath.Join(path, "DlrTracer.data"), &t.data)
		t.heapSnap = ssnap.New(filepath.Join(path, "DlrTracer.heap"), &t.heap)
	}
	return t
}

func (t *DlrTracer) Save(_ bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.dataSnap.Save(); err != nil {
		return err
	}

	if err := t.heapSnap.Save(); err != nil {
		return err
	}

	return nil
}

func (t *DlrTracer) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.dataSnap.Load(); err != nil {
		return err
	}

	if err := t.heapSnap.Load(); err != nil {
		return err
	}

	return nil
}

func (t *DlrTracer) Put(to *DlrEntry) {
	t.mu.Lock()
	t.data[to.MessageId] = to
	t.heap.Push(to)
	t.mu.Unlock()
}

func (t *DlrTracer) Take(messageId string) *DlrEntry {
	t.mu.Lock()
	to, ok := t.data[messageId]
	if ok {
		to.hasTook = true
		delete(t.data, messageId)
	}
	t.mu.Unlock()

	return to
}

func (t *DlrTracer) TakeTimeout() []*DlrEntry {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.heap.Len() == 0 {
		return nil
	}

	list := make([]*DlrEntry, 0, 32)
	curr := time.Now().Unix()
	for len(t.heap) > 0 {
		to := t.heap[0]
		if !to.hasTook {
			if curr < to.ExpiredAt {
				break
			}
			list = append(list, to)
			to.hasTook = true
			delete(t.data, to.MessageId)
		}
		heap.Pop(&t.heap)
	}

	return list
}
