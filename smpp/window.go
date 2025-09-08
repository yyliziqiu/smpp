package smpp

import (
	"sync"
	"time"

	"github.com/yyliziqiu/slib/scq2"
)

type Window interface {
	Put(*TRequest) error
	Take(int32) *TRequest
	TakeTimeout() []*TRequest
}

// ============ MapWindow ============

type MapWindow struct {
	size int                 // 窗口大小
	wait int64               // 请求超时时间
	data map[int32]*TRequest //
	mu   sync.Mutex          //
}

func NewMapWindow(size int, wait time.Duration) Window {
	return &MapWindow{
		size: size,
		wait: int64(wait.Seconds()),
		data: make(map[int32]*TRequest, size),
	}
}

func (w *MapWindow) Put(request *TRequest) error {
	w.mu.Lock()
	err := w.put(request)
	// logDebug("[MapWindow] Put request, sequence: %d, ok: %v", request.Pdu.GetSequenceNumber(), err == nil)
	// logDebug("[MapWindow] Data: %v", w.data)
	w.mu.Unlock()

	return err
}

func (w *MapWindow) put(request *TRequest) error {
	if len(w.data) >= w.size {
		return ErrWindowFull
	}

	w.data[request.Pdu.GetSequenceNumber()] = request

	return nil
}

func (w *MapWindow) Take(sequence int32) *TRequest {
	w.mu.Lock()
	request, ok := w.data[sequence]
	if ok {
		delete(w.data, sequence)
	}
	// logDebug("[MapWindow] Take request, sequence: %d, ok: %v", sequence, request != nil)
	// logDebug("[MapWindow] Data: %v", w.data)
	w.mu.Unlock()

	return request
}

func (w *MapWindow) TakeTimeout() []*TRequest {
	requests := make([]*TRequest, 0, w.size/5)

	w.mu.Lock()
	curr := time.Now().Unix()
	for seq, request := range w.data {
		if curr-w.wait > request.SubmitAt {
			delete(w.data, seq)
			requests = append(requests, request)
		}
	}
	// logDebug("[MapWindow] Take timeout requests, count: %d", len(requests))
	// logDebug("[MapWindow] Data: %v", w.data)
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
	Request *TRequest
}

func NewQueueWindow(size int, wait time.Duration) Window {
	return &QueueWindow{
		size:  size,
		wait:  int64(wait.Seconds()),
		data:  make(map[int32]*QueueWindowValue, size),
		queue: scq2.New(size * 2),
	}
}

func (w *QueueWindow) Put(request *TRequest) error {
	w.mu.Lock()
	err := w.put(request)
	// logDebug("[QueueWindow] Put request, sequence: %d, ok: %v", request.Pdu.GetSequenceNumber(), err == nil)
	// logDebug("[QueueWindow] Data: %v", w.data)
	w.mu.Unlock()

	return err
}

func (w *QueueWindow) put(request *TRequest) error {
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

func (w *QueueWindow) Take(sequence int32) *TRequest {
	w.mu.Lock()
	request := w.take(sequence)
	// logDebug("[QueueWindow] Take request, sequence: %d, ok: %v", sequence, request != nil)
	// logDebug("[QueueWindow] Data: %v", w.data)
	w.mu.Unlock()

	return request
}

func (w *QueueWindow) take(sequence int32) *TRequest {
	value, ok := w.data[sequence]
	if !ok {
		return nil
	}

	delete(w.data, sequence)

	request := value.Request

	value.Request = nil

	return request
}

func (w *QueueWindow) TakeTimeout() []*TRequest {
	list := make([]*TRequest, 0, w.size/5)
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
	// logDebug("[QueueWindow] Take timeout requests, count: %d", len(list))
	// logDebug("[QueueWindow] Data: %v", w.data)
	w.mu.Unlock()

	return list
}
