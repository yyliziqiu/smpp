package smpp

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/slib/stime"
	"github.com/yyliziqiu/slib/suid"
)

type Session struct {
	id     string
	conn   Connection
	conf   *SessionConfig
	term   *SessionTerm
	tracer *Tracer
	status int32
	closed int32
	initAt time.Time
}

type SessionTerm struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	window Window
	trChan chan *TRequest
	dialAt time.Time
}

type SessionConfig struct {
	EnquireLink    time.Duration                       // 心跳间隔
	AttemptDial    time.Duration                       // 重连间隔
	WindowClear    time.Duration                       // 清理窗口内超时请求的时间间隔
	WindowType     int                                 // 窗口类型，WinTimeout 小或 WinCapacity 大时建议为1
	WindowSize     int                                 // 窗口大小
	WindowWait     time.Duration                       // 超时时间
	CustomData     any                                 // 自定义数据
	OnCreateWindow func(string, any) Window            // 根据 system id 创建窗口
	OnReceive      func(*RRequest, any) pdu.PDU        // 接收到对端非响应 pdu 时执行
	OnRequest      func(*TRequest, any)                // 向对端提交 pdu 时执行
	OnRespond      func(*TResponse, any)               // 接收到对端 pdu 响应时执行，此响应为 OnRequest 提交的 pdu 的响应
	OnCreated      func(*Session, any)                 // 创建会话时执行
	OnClosed       func(*Session, string, string, any) // 关闭会话时执行
}

func NewSession(conn Connection, conf SessionConfig) (*Session, error) {
	if conf.WindowClear == 0 {
		conf.WindowClear = time.Minute
	}
	if conf.WindowSize == 0 {
		conf.WindowSize = 100
	}
	if conf.WindowWait == 0 {
		conf.WindowWait = 10 * time.Minute
	}

	s := &Session{
		id:     suid.Get(),
		conn:   conn,
		conf:   &conf,
		term:   nil,
		tracer: _tracer,
		status: SessionClosed,
		closed: 0,
		initAt: time.Now(),
	}

	err := s.dial()
	if err != nil {
		return nil, err
	}

	s.onCreated(s.customData())

	return s, nil
}

func (s *Session) dial() error {
	if atomic.LoadInt32(&s.status) == SessionActive {
		return nil
	}

	err := s.conn.Dial()
	if err != nil {
		logWarn("[Session@%s:%s] Dial failed, peer addr: %s, error: %v", s.id, s.SystemId(), s.PeerAddr(), err)
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.term = &SessionTerm{
		wg:     sync.WaitGroup{},
		ctx:    ctx,
		cancel: cancel,
		window: s.createWindow(),
		trChan: make(chan *TRequest, 1),
		dialAt: time.Now(),
	}

	s.term.wg.Add(3)

	atomic.StoreInt32(&s.status, SessionActive)

	s.loopRead()
	s.loopWrite()
	s.loopWindow()

	logInfo("[Session@%s:%s] Dial succeed, peer addr: %s", s.id, s.SystemId(), s.PeerAddr())

	return nil
}

func (s *Session) createWindow() Window {
	if s.conf.OnCreateWindow != nil {
		return s.conf.OnCreateWindow(s.SystemId(), s.customData())
	}

	switch s.conf.WindowType {
	case 1:
		return NewQueueWindow(s.conf.WindowSize, s.conf.WindowWait)
	default:
		return NewMapWindow(s.conf.WindowSize, s.conf.WindowWait)
	}
}

func (s *Session) customData() any {
	return s.conf.CustomData
}

func (s *Session) loopRead() {
	go func() {
		defer s.term.wg.Done()
		for {
			select {
			case <-s.term.ctx.Done():
				logDebug("[Session@%s:%s] Loop read exit", s.id, s.SystemId())
				return
			default:
				if s.read() {
					logDebug("[Session@%s:%s] Loop read stop", s.id, s.SystemId())
					return
				}
			}
		}
	}()
}

func (s *Session) read() bool {
	p, err := s.conn.Read()
	if err != nil {
		logWarn("[Session@%s:%s] Read failed, error: %v", s.id, s.SystemId(), err)
		s.close(CloseByError, err.Error())
		return true
	}

	if !s.allowRead(p) {
		return false
	}

	switch p.(type) {
	case *pdu.EnquireLink:
		logDebug("[Session@%s:%s] Received enquire link pdu", s.id, s.SystemId())
		s.writeToQueue(p.GetResponse(), SubmitterSys)
		return false
	case *pdu.EnquireLinkResp:
		s.term.window.Take(p.GetSequenceNumber())
		return false
	case *pdu.Unbind:
		logInfo("[Session@%s:%s] Received unbind pdu", s.id, s.SystemId())
		s.writeToQueue(p.GetResponse(), SubmitterSys)
		s.close(CloseByPdu, "received unbind")
		return true
	case *pdu.UnbindResp:
		logInfo("[Session@%s:%s] Received unbind resp pdu", s.id, s.SystemId())
		s.close(CloseByPdu, "received unbind response")
		return true
	case *pdu.BindRequest:
		return false
	case *pdu.AlertNotification:
		s.onReceive(NewRRequest(s, p), s.customData())
		return false
	case *pdu.GenericNack, *pdu.Outbind:
		logInfo("[Session@%s:%s] Received generic nack or out bind pdu", s.id, s.SystemId())
		s.close(CloseByPdu, "received unexpected pdu")
		return true
	}

	// AlertNotification, Outbind, GenericNack 这3类 pdu 没有对应的 resp
	if p.CanResponse() {
		rp := s.onReceive(NewRRequest(s, p), s.customData())
		if rp != nil {
			s.writeToQueue(rp, SubmitterSys)
		}
	} else {
		tr := s.term.window.Take(p.GetSequenceNumber())
		if tr != nil {
			s.onRespond(NewTResponse(s, tr, p, nil), s.customData())
		}
	}

	return false
}

func (s *Session) close(reason string, desc string) {
	if !atomic.CompareAndSwapInt32(&s.status, SessionActive, SessionClosed) {
		return
	}

	go func() {
		logInfo("[Session@%s:%s] Closing, reason: %s, desc: %s", s.id, s.SystemId(), reason, desc)

		// 让正在阻塞中的读写操作超时退出
		_ = s.conn.SetDeadline(time.Now().Add(300 * time.Millisecond))

		// 停止读写协程
		s.term.cancel()

		// 等待读写协程停止
		s.term.wg.Wait()

		// 关闭链接
		_ = s.conn.Close()

		// 清理会话数据
		close(s.term.trChan)
		for request := range s.term.trChan {
			s.onRespond(NewTResponse(s, request, nil, ErrChannelClosed), s.customData())
		}
		s.term.window = nil

		logInfo("[Session@%s:%s] Closed", s.id, s.SystemId())

		// 结束会话
		closed := s.conf.AttemptDial == 0 || reason == CloseByExplicit
		if closed {
			s.onClosed(reason, desc, s.customData())
			return
		}

		logInfo("[Session@%s:%s] Redialing", s.id, s.SystemId())

		// 重新启动会话
		ticker := time.NewTicker(s.conf.AttemptDial)
		defer ticker.Stop()
		for {
			<-ticker.C
			if atomic.LoadInt32(&s.closed) == 1 {
				logInfo("[Session@%s:%s] Close when redialing", s.id, s.SystemId())
				s.onClosed(CloseByExplicit, "", s.customData())
				return
			}
			err := s.dial()
			if err == nil {
				if atomic.LoadInt32(&s.closed) == 1 {
					logInfo("[Session@%s:%s] Close when redialed", s.id, s.SystemId())
					s.close(CloseByExplicit, "")
				}
				return
			}
		}
	}()
}

func (s *Session) allowRead(p pdu.PDU) bool {
	// todo 根据 session 角色和绑定类型限制可以接收哪些类型的 pdu
	return true
}

func (s *Session) writeToQueue(p pdu.PDU, submitter int8) {
	s.term.trChan <- s.newTRequest(p, submitter)
}

func (s *Session) newTRequest(p pdu.PDU, submitter int8) *TRequest {
	return &TRequest{
		submitter: submitter,
		SystemId:  s.conn.SystemId(),
		SessionId: s.id,
		MessageId: suid.Get(),
		CreateAt:  time.Now().Unix(),
		SubmitAt:  0,
		Pdu:       p,
	}
}

func (s *Session) loopWrite() {
	if s.conf.EnquireLink == 0 {
		go func() {
			defer s.term.wg.Done()
			for {
				select {
				case <-s.term.ctx.Done():
					s.logLoopWriteExit()
					return
				case r := <-s.term.trChan:
					if s.write(r) {
						s.logLoopWriteStop()
						return
					}
				}
			}
		}()
	} else {
		go func() {
			t := time.NewTicker(s.conf.EnquireLink)
			defer func() {
				t.Stop()
				s.term.wg.Done()
			}()
			for {
				select {
				case <-s.term.ctx.Done():
					s.logLoopWriteExit()
					return
				case r := <-s.term.trChan:
					if s.write(r) {
						s.logLoopWriteStop()
						return
					}
				case <-t.C:
					if s.write(s.newTRequest(pdu.NewEnquireLink(), SubmitterSys)) {
						s.logLoopWriteStop()
						return
					}
				}
			}
		}()
	}
}

func (s *Session) logLoopWriteExit() {
	logDebug("[Session@%s:%s] Loop write exit", s.id, s.SystemId())
}

func (s *Session) logLoopWriteStop() {
	logDebug("[Session@%s:%s] Loop write stop", s.id, s.SystemId())
}

func (s *Session) write(request *TRequest) bool {
	if request == nil {
		return true
	}

	if s.closed == 1 {
		s.onRespond(NewTResponse(s, request, nil, ErrSessionClosed), s.customData())
		return true
	}

	if !s.allowWrite(request.Pdu) {
		s.onRespond(NewTResponse(s, request, nil, ErrNotAllowed), s.customData())
		return false
	}

	request.SubmitAt = time.Now().Unix()
	s.onRequest(request, s.customData())

	if request.Pdu.CanResponse() {
		err := s.term.window.Put(request)
		if err != nil {
			logWarn("[Session@%s:%s] Put request to window failed, error: %v", s.id, s.SystemId(), err)
			s.onRespond(NewTResponse(s, request, nil, err), s.customData())
			return false
		}
	}

	n, err := s.conn.Write(request.Pdu)
	if err != nil {
		logWarn("[Session@%s:%s] Write failed, error: %v", s.id, s.SystemId(), err)
		s.onRespond(NewTResponse(s, request, nil, err), s.customData())
		if n > 0 {
			s.close(CloseByError, err.Error())
			return true
		} else {
			if nerr, ok := err.(net.Error); !ok || !nerr.Timeout() {
				s.close(CloseByError, err.Error())
				return true
			}
		}
	}

	return false
}

func (s *Session) allowWrite(p pdu.PDU) bool {
	switch p.(type) {
	case *pdu.BindRequest, *pdu.Unbind, *pdu.Outbind, *pdu.GenericNack, *pdu.AlertNotification:
		return false
	}
	// todo 根据 session 角色和绑定类型限制可以提交哪些类型的 pdu
	return true
}

func (s *Session) loopWindow() {
	go func() {
		t := time.NewTicker(s.conf.WindowClear)
		defer func() {
			t.Stop()
			s.term.wg.Done()
		}()
		for {
			select {
			case <-s.term.ctx.Done():
				logDebug("[Session@%s:%s] Loop window exit", s.id, s.SystemId())
				return
			case <-t.C:
				timer := stime.NewTimer()
				requests := s.term.window.TakeTimeout()
				for _, request := range requests {
					s.onRespond(NewTResponse(s, request, nil, ErrResponseTimeout), s.customData())
				}
				logDebug("[Session@%s:%s] Handled timeout requests, count: %d, cost: %s", s.id, s.SystemId(), len(requests), timer.Stops())
			}
		}
	}()
}

func (s *Session) onReceive(request *RRequest, values any) pdu.PDU {
	if s.conf.OnReceive != nil {
		return s.conf.OnReceive(request, values)
	}
	return nil
}

func (s *Session) onRequest(request *TRequest, values any) {
	if s.conf.OnRequest != nil && request.submitter == SubmitterUser {
		s.conf.OnRequest(request, values)
	}
}

func (s *Session) onRespond(response *TResponse, values any) {
	if s.conf.OnRespond != nil && response.Request.submitter == SubmitterUser {
		s.conf.OnRespond(response, values)
	}
}

func (s *Session) onCreated(values any) {
	s.tracer.AddSession(s)
	if s.conf.OnCreated != nil {
		s.conf.OnCreated(s, values)
	}
}

func (s *Session) onClosed(reason string, desc string, values any) {
	s.tracer.DelSession(s.id)
	if s.conf.OnClosed != nil {
		s.conf.OnClosed(s, reason, desc, values)
	}
}

func (s *Session) Id() string {
	return s.id
}

func (s *Session) Status() int32 {
	return atomic.LoadInt32(&s.status)
}

func (s *Session) SystemId() string {
	return s.conn.SystemId()
}

func (s *Session) BindType() pdu.BindingType {
	return s.conn.BindType()
}

func (s *Session) PeerAddr() string {
	return s.conn.PeerAddr()
}

func (s *Session) InitAt() time.Time {
	return s.initAt
}

func (s *Session) DialAt() time.Time {
	return s.term.dialAt
}

func (s *Session) Write(p pdu.PDU) error {
	if atomic.LoadInt32(&s.status) == SessionClosed {
		return ErrSessionClosed
	}
	s.writeToQueue(p, SubmitterUser)
	return nil
}

func (s *Session) Close() {
	atomic.StoreInt32(&s.closed, 1)
	s.close(CloseByExplicit, "")
}

func (s *Session) Closed() bool {
	return atomic.LoadInt32(&s.status) == SessionClosed
}

func (s *Session) ClosedExplicitly() bool {
	return atomic.LoadInt32(&s.closed) == 1
}
