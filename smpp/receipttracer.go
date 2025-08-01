package smpp

import (
	"container/heap"
	"path/filepath"
	"sync"
	"time"

	"github.com/yyliziqiu/slib/ssnap"
)

type ReceiptTracer struct {
	data     map[string]*ReceiptTo
	heap     ReceiptHeap
	dataSnap *ssnap.Snap
	heapSnap *ssnap.Snap
	mu       sync.Mutex
}

func NewReceiptTracer(size int) *ReceiptTracer {
	return NewReceiptTracerWithSnap(size, "")
}

func NewReceiptTracerWithSnap(size int, path string) *ReceiptTracer {
	t := &ReceiptTracer{
		data: make(map[string]*ReceiptTo, size),
		heap: make(ReceiptHeap, 0, size),
	}
	if path != "" {
		t.dataSnap = ssnap.New(filepath.Join(path, "ReceiptTracer.data"), &t.data)
		t.heapSnap = ssnap.New(filepath.Join(path, "ReceiptTracer.heap"), &t.heap)
	}
	return t
}

func (t *ReceiptTracer) Save(_ bool) error {
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

func (t *ReceiptTracer) Load() error {
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

func (t *ReceiptTracer) Put(to *ReceiptTo) {
	t.mu.Lock()
	t.data[to.MessageId] = to
	t.heap.Push(to)
	t.mu.Unlock()
}

func (t *ReceiptTracer) Take(messageId string) *ReceiptTo {
	t.mu.Lock()
	to, ok := t.data[messageId]
	if ok {
		to.hasTook = true
		delete(t.data, messageId)
	}
	t.mu.Unlock()

	return to
}

func (t *ReceiptTracer) TakeTimeout() []*ReceiptTo {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.heap.Len() == 0 {
		return nil
	}

	list := make([]*ReceiptTo, 0, 32)
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
