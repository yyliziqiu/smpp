package smpp

import (
	"sync"
	"time"

	"github.com/yyliziqiu/gdk/xcq"
	"golang.org/x/exp/maps"
)

type Window interface {
	Full() bool
	Data() map[int32]*Request
	Put(*Request) error
	Take(int32) *Request
	TakeTimeout() []*Request
}

func CreateWindow(sess *Session) Window {
	switch sess.conf.WindowType {
	case 1:
		return NewLargeWindow(sess.conf.WindowSize, sess.conf.WindowWait)
	default:
		return NewSmallWindow(sess.conf.WindowSize, sess.conf.WindowWait)
	}
}

// ============ SmallWindow ============

type SmallWindow struct {
	size int                // 窗口大小
	wait int64              // 请求超时时间
	data map[int32]*Request //
	mu   sync.Mutex         //
}

func NewSmallWindow(size int, wait time.Duration) Window {
	return &SmallWindow{
		size: size,
		wait: int64(wait.Seconds()),
		data: make(map[int32]*Request, size),
	}
}

func (w *SmallWindow) Full() bool {
	w.mu.Lock()
	full := w.full()
	w.mu.Unlock()

	return full
}

func (w *SmallWindow) full() bool {
	return len(w.data) >= w.size
}

func (w *SmallWindow) Data() map[int32]*Request {
	w.mu.Lock()
	requests := maps.Clone(w.data)
	w.mu.Unlock()

	return requests
}

func (w *SmallWindow) Put(request *Request) error {
	w.mu.Lock()
	err := w.put(request)
	w.mu.Unlock()

	return err
}

func (w *SmallWindow) put(request *Request) error {
	if w.full() {
		return ErrWindowFull
	}

	w.data[request.Pdu.GetSequenceNumber()] = request

	return nil
}

func (w *SmallWindow) Take(sequence int32) *Request {
	w.mu.Lock()
	request, ok := w.data[sequence]
	if ok {
		delete(w.data, sequence)
	}
	w.mu.Unlock()

	return request
}

func (w *SmallWindow) TakeTimeout() []*Request {
	requests := make([]*Request, 0, 8)

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

// ============ LargeWindow ============

type LargeWindow struct {
	size  int
	wait  int64
	data  map[int32]*LargeWindowValue
	queue *xcq.Queue
	mu    sync.Mutex
}

type LargeWindowValue struct {
	Request *Request
}

func NewLargeWindow(size int, wait time.Duration) Window {
	return &LargeWindow{
		size:  size,
		wait:  int64(wait.Seconds()),
		data:  make(map[int32]*LargeWindowValue, size),
		queue: xcq.New(size * 2),
	}
}

func (w *LargeWindow) Full() bool {
	w.mu.Lock()
	full := w.full()
	w.mu.Unlock()

	return full
}

func (w *LargeWindow) full() bool {
	return len(w.data) >= w.size
}

func (w *LargeWindow) Data() map[int32]*Request {
	requests := make(map[int32]*Request, len(w.data))

	w.mu.Lock()
	for key, value := range w.data {
		requests[key] = value.Request
	}
	w.mu.Unlock()

	return requests
}

func (w *LargeWindow) Put(request *Request) error {
	w.mu.Lock()
	err := w.put(request)
	w.mu.Unlock()

	return err
}

func (w *LargeWindow) put(request *Request) error {
	if w.full() {
		return ErrWindowFull
	}

	value := &LargeWindowValue{
		Request: request,
	}

	w.data[request.Pdu.GetSequenceNumber()] = value
	w.queue.Push(value)

	return nil
}

func (w *LargeWindow) Take(sequence int32) *Request {
	w.mu.Lock()
	request := w.take(sequence)
	w.mu.Unlock()

	return request
}

func (w *LargeWindow) take(sequence int32) *Request {
	value, ok := w.data[sequence]
	if !ok {
		return nil
	}

	delete(w.data, sequence)

	request := value.Request
	value.Request = nil

	return request
}

func (w *LargeWindow) TakeTimeout() []*Request {
	curr := time.Now().Unix()
	list := make([]*Request, 0, 16)

	w.mu.Lock()
	w.queue.Pops2(func(item any) bool {
		value := item.(*LargeWindowValue)
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
