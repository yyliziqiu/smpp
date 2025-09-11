package smpp

import (
	"container/heap"
	"path/filepath"
	"sync"
	"time"

	"github.com/yyliziqiu/slib/ssnap"
)

type DlrItem struct {
	MessageId string
	SystemId  string
	SessionId string
	ExpiredAt int64
	hasTook   bool
}

type DlrHeap []*DlrItem

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
	*h = append(*h, v.(*DlrItem))
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
	data  map[string]*DlrItem
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
		data: make(map[string]*DlrItem, size),
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

func (t *DlrTracer) Put(item *DlrItem) {
	t.mu.Lock()
	t.data[item.MessageId] = item
	t.heap.Push(item)
	t.mu.Unlock()
}

func (t *DlrTracer) Take(messageId string) *DlrItem {
	t.mu.Lock()
	item, ok := t.data[messageId]
	if ok {
		item.hasTook = true
		delete(t.data, messageId)
	}
	t.mu.Unlock()

	return item
}

func (t *DlrTracer) TakeTimeout() []*DlrItem {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.heap.Len() == 0 {
		return nil
	}

	list := make([]*DlrItem, 0, 32)
	curr := time.Now().Unix()
	for len(t.heap) > 0 {
		item := t.heap[0]
		if !item.hasTook {
			if curr < item.ExpiredAt {
				break
			}
			list = append(list, item)
			item.hasTook = true
			delete(t.data, item.MessageId)
		}
		heap.Pop(&t.heap)
	}

	return list
}
