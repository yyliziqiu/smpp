package smpp

import (
	"sync"
	"time"

	"github.com/yyliziqiu/slib/scq2"
)

type Window interface {
	Put(*Request) error
	Take(int32) *Request
	TakeTimeout() []*Request
}

// ============ MapWindow ============

type MapWindow struct {
	size int                // 窗口大小
	wait int64              // 请求超时时间
	data map[int32]*Request //
	mu   sync.Mutex         //
}

func NewMapWindow(size int, wait time.Duration) Window {
	return &MapWindow{
		size: size,
		wait: int64(wait.Seconds()),
		data: make(map[int32]*Request, size),
	}
}

func (w *MapWindow) Put(request *Request) error {
	w.mu.Lock()
	err := w.put(request)
	w.mu.Unlock()

	return err
}

func (w *MapWindow) put(request *Request) error {
	if len(w.data) >= w.size {
		return ErrWindowFull
	}

	w.data[request.Pdu.GetSequenceNumber()] = request

	return nil
}

func (w *MapWindow) Take(sequence int32) *Request {
	w.mu.Lock()
	request, ok := w.data[sequence]
	if ok {
		delete(w.data, sequence)
	}
	w.mu.Unlock()

	return request
}

func (w *MapWindow) TakeTimeout() []*Request {
	requests := make([]*Request, 0, w.size/5)

	w.mu.Lock()
	curr := time.Now().Unix()
	for seq, request := range w.data {
		if curr-w.wait > request.SubmitAt {
			delete(w.data, seq)
			requests = append(requests, request)
		}
	}
	w.mu.Unlock()

	return requests
}

// ============ QueueWindow ============

type QueueWindow struct {
	size  int
	wait  int64
	data  map[int32]*QueueWindowValue
	queue *scq2.Queue
	mu    sync.Mutex
}

type QueueWindowValue struct {
	Request *Request
}

func NewQueueWindow(size int, wait time.Duration) Window {
	return &QueueWindow{
		size:  size,
		wait:  int64(wait.Seconds()),
		data:  make(map[int32]*QueueWindowValue, size),
		queue: scq2.New(size * 2),
	}
}

func (w *QueueWindow) Put(request *Request) error {
	w.mu.Lock()
	err := w.put(request)
	w.mu.Unlock()

	return err
}

func (w *QueueWindow) put(request *Request) error {
	if len(w.data) >= w.size {
		return ErrWindowFull
	}

	value := &QueueWindowValue{
		Request: request,
	}

	w.data[request.Pdu.GetSequenceNumber()] = value

	w.queue.Push(value)

	return nil
}

func (w *QueueWindow) Take(sequence int32) *Request {
	w.mu.Lock()
	request := w.take(sequence)
	w.mu.Unlock()

	return request
}

func (w *QueueWindow) take(sequence int32) *Request {
	value, ok := w.data[sequence]
	if !ok {
		return nil
	}

	delete(w.data, sequence)

	request := value.Request

	value.Request = nil

	return request
}

func (w *QueueWindow) TakeTimeout() []*Request {
	list := make([]*Request, 0, w.size/5)
	curr := time.Now().Unix()

	w.mu.Lock()
	w.queue.Pops2(func(item any) bool {
		value := item.(*QueueWindowValue)
		if value.Request == nil {
			return true
		}
		if curr-w.wait > value.Request.SubmitAt {
			delete(w.data, value.Request.Pdu.GetSequenceNumber())
			list = append(list, value.Request)
			return true
		}
		return false
	})
	w.mu.Unlock()

	return list
}
