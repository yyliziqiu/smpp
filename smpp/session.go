package smpp

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/sirupsen/logrus"
	"github.com/yyliziqiu/gdk/xuid"
)

const (
	ConnectionClosed = 0
	ConnectionDialed = 1

	SubmitterSys  = 1
	SubmitterUser = 2

	SessionDialing = "dialing"
	SessionActive  = "active"
	SessionClosed  = "closed"

	CloseByError    = "error"
	CloseByPdu      = "pdu"
	CloseByExplicit = "explicit"
)

type Session struct {
	id     string         //
	slog   *logrus.Logger //
	store  *SessionStore  //
	conn   Connection     //
	conf   *SessionConfig //
	term   *SessionTerm   //
	status int32          // 连接状态
	closed int32          // 会话是否被显示关闭
	initAt time.Time      // 会话创建时间
}

type SessionTerm struct {
	swg    sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	window Window
	sendCh chan *Request
	dialAt time.Time
}

type SessionConfig struct {
	Context     any
	EnquireLink time.Duration                   // 心跳间隔
	AttemptDial time.Duration                   // 重连间隔
	WindowType  int                             // 窗口类型，WindowWait 小或 WindowSize 大时建议为1
	WindowSize  int                             // 窗口大小
	WindowWait  time.Duration                   // 超时时间
	WindowScan  time.Duration                   // 清理窗口内超时请求的时间间隔
	WindowBlock time.Duration                   // 当窗口满时写操作的阻塞时间，0表示不阻塞返回错误，大于0表示阻塞时间，小于0表示挂起当前协程等待下次调度
	WindowNewer func(*Session) Window           // 自定义窗口
	OnDialed    func(*Session)                  // 连接成功时执行
	OnClosed    func(*Session, string, string)  // 关闭会话时执行
	OnReceive   func(*Session, pdu.PDU) pdu.PDU // 接收到对端非响应 pdu 时执行
	OnRequest   func(*Session, *Request)        // 向对端提交 pdu 时执行
	OnRespond   func(*Session, *Response)       // 接收到对端 pdu 响应时执行，此响应为 OnRequest 提交的 pdu 的响应
}

func NewSession(conn Connection, conf SessionConfig) (*Session, error) {
	if conf.WindowSize == 0 {
		conf.WindowSize = 64
	}
	if conf.WindowWait == 0 {
		conf.WindowWait = time.Minute
	}
	if conf.WindowScan == 0 {
		conf.WindowScan = time.Minute
	}

	// 创建 session
	s := &Session{
		id:     xuid.Get(),
		slog:   _slog,
		store:  _store,
		conn:   conn,
		conf:   &conf,
		term:   nil,
		status: ConnectionClosed,
		closed: 0,
		initAt: time.Now(),
	}

	// 连接
	err := s.dial()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Session) dial() error {
	if s.connDialed() {
		return nil
	}

	if err := s.conn.Dial(); err != nil {
		s.warn("Dial failed, peer addr: %s, error: %v", s.PeerAddr(), err)
		return err
	}

	if s.conf.WindowNewer == nil {
		s.conf.WindowNewer = CreateWindow
	}

	ctx, cancel := context.WithCancel(context.Background())

	s.term = &SessionTerm{
		swg:    sync.WaitGroup{},
		ctx:    ctx,
		cancel: cancel,
		window: s.conf.WindowNewer(s),
		sendCh: make(chan *Request, 1),
		dialAt: time.Now(),
	}
	s.term.swg.Add(4)

	atomic.StoreInt32(&s.status, ConnectionDialed)

	s.loopRead()
	s.loopSend()
	s.loopWrite()
	s.loopClear()

	s.onDialed()
	s.info("Dial succeed, peer addr: %s", s.PeerAddr())

	return nil
}

func (s *Session) connDialed() bool {
	return atomic.LoadInt32(&s.status) == ConnectionDialed
}

func (s *Session) connClosed() bool {
	return atomic.LoadInt32(&s.status) == ConnectionClosed
}

func (s *Session) close(reason string, desc string) {
	if !atomic.CompareAndSwapInt32(&s.status, ConnectionDialed, ConnectionClosed) {
		return
	}

	go func() {
		s.info("Closing, reason: %s, desc: %s", reason, desc)

		// 停止读写协程
		s.term.cancel()
		time.Sleep(100 * time.Millisecond)

		// 让正在阻塞中的读写操作超时退出
		_ = s.conn.Deadline(time.Now().Add(100 * time.Millisecond))

		// 等待读写协程停止
		s.term.swg.Wait()

		// 关闭链接
		_ = s.conn.Close(reason == CloseByExplicit)

		// 清理会话数据
		close(s.term.sendCh)
		for request := range s.term.sendCh {
			s.onRespond(NewResponse(request, nil, ErrChannelClosed))
		}
		s.term.window = nil

		s.info("Closed")

		// 结束会话
		closed := s.conf.AttemptDial == 0 || reason == CloseByExplicit
		if closed {
			s.onClosed(reason, desc)
			return
		}

		s.info("Redialing")

		// 重新启动会话
		ticker := time.NewTicker(s.conf.AttemptDial)
		defer ticker.Stop()
		for {
			<-ticker.C
			if atomic.LoadInt32(&s.closed) == 1 {
				s.info("Close when redialing")
				s.onClosed(CloseByExplicit, "")
				return
			}
			err := s.dial()
			if err == nil {
				if atomic.LoadInt32(&s.closed) == 1 {
					s.info("Close when redialed")
					s.close(CloseByExplicit, "")
				}
				return
			}
		}
	}()
}

func (s *Session) loopRead() {
	go func() {
		defer s.term.swg.Done()
		for {
			select {
			case <-s.term.ctx.Done():
				return
			default:
				if s.read() {
					return
				}
			}
		}
	}()
}

func (s *Session) read() bool {
	p, err := s.conn.Read()
	if err != nil {
		s.warn("Read failed, error: %v", err)
		s.close(CloseByError, err.Error())
		return true
	}

	if !s.allowRead(p) {
		return false
	}

	switch p.(type) {
	case *pdu.EnquireLink:
		s.debug("Received enquire link pdu")
		s.writeToQueue(SubmitterSys, p.GetResponse(), nil)
		return false
	case *pdu.EnquireLinkResp:
		s.term.window.Take(p.GetSequenceNumber())
		return false
	case *pdu.Unbind:
		s.info("Received unbind pdu")
		s.writeToQueue(SubmitterSys, p.GetResponse(), nil)
		s.close(CloseByPdu, "received unbind pdu")
		return true
	case *pdu.UnbindResp:
		s.info("Received unbind resp pdu")
		s.close(CloseByPdu, "received unbind response pdu")
		return true
	case *pdu.BindRequest:
		return false
	case *pdu.AlertNotification:
		s.onReceive(p)
		return false
	case *pdu.GenericNack, *pdu.Outbind:
		s.info("Received generic nack or out bind pdu")
		s.close(CloseByPdu, "received unexpected pdu")
		return true
	}

	// AlertNotification, Outbind, GenericNack 这3类 pdu 没有对应的 resp
	if p.CanResponse() {
		rp := s.onReceive(p)
		if rp != nil {
			s.writeToQueue(SubmitterSys, rp, nil)
		}
	} else {
		tr := s.term.window.Take(p.GetSequenceNumber())
		if tr != nil {
			s.onRespond(NewResponse(tr, p, nil))
		}
	}

	return false
}

func (s *Session) allowRead(_ pdu.PDU) bool {
	// todo 根据会话角色和绑定类型限制可以接收哪些类型的 pdu
	return true
}

func (s *Session) writeToQueue(submitter int8, p pdu.PDU, data any) {
	s.term.sendCh <- s.newRequest(submitter, p, data)
}

func (s *Session) newRequest(submitter int8, p pdu.PDU, data any) *Request {
	return &Request{
		Pdu:       p,
		TraceData: data,
		SessionId: s.id,
		SystemId:  s.conn.SystemId(),
		SubmitAt:  0,
		submitter: submitter,
	}
}

func (s *Session) loopSend() {

}

func (s *Session) loopWrite() {
	go func() {
		defer s.term.swg.Done()
		if s.conf.EnquireLink == 0 {
			for {
				select {
				case <-s.term.ctx.Done():
					return
				case r := <-s.term.sendCh:
					if s.write(r) {
						return
					}
				}
			}
		} else {
			tk := time.NewTicker(s.conf.EnquireLink)
			defer tk.Stop()
			for {
				select {
				case <-s.term.ctx.Done():
					return
				case r := <-s.term.sendCh:
					if s.write(r) {
						return
					}
				case <-tk.C:
					if s.write(s.newRequest(SubmitterSys, pdu.NewEnquireLink(), nil)) {
						return
					}
				}
			}
		}
	}()
}

func (s *Session) write(request *Request) bool {
	if request == nil {
		return true
	}

	// 若链接已关闭，则尽快结束此协程
	if s.connClosed() {
		s.onRespond(NewResponse(request, nil, ErrConnectionClosed))
		return true
	}

	// 判断 pdu 是否可以发送
	if !s.allowWrite(request.Pdu) {
		s.onRespond(NewResponse(request, nil, ErrNotAllowed))
		return false
	}

	// 可以响应的 pdu 需要添加到窗口中
	request.SubmitAt = time.Now().Unix()
	if request.Pdu.CanResponse() {
		// 若窗口已满，则等待窗口可用
		if s.conf.WindowBlock != 0 {
			for s.term.window.Full() {
				if s.connClosed() { // 防止此协程不能退出
					s.onRespond(NewResponse(request, nil, ErrConnectionClosed))
					return true
				}
				if s.conf.WindowBlock > 0 {
					time.Sleep(s.conf.WindowBlock)
				} else {
					runtime.Gosched()
				}
			}
		}
		// 将请求添加至窗口
		err := s.term.window.Put(request)
		if err != nil {
			s.warn("Put request to window failed, error: %v", err)
			s.onRespond(NewResponse(request, nil, err))
			return false
		}
	}

	// 执行回调
	s.onRequest(request)

	// 发送 pdu
	n, err := s.conn.Write(request.Pdu)
	if err != nil {
		s.warn("Write failed, error: %v", err)
		s.onRespond(NewResponse(request, nil, err))
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
	// todo 根据会话角色和绑定类型限制可以提交哪些类型的 pdu
	return true
}

func (s *Session) loopClear() {
	go func() {
		defer s.term.swg.Done()
		tk := time.NewTicker(s.conf.WindowScan)
		defer tk.Stop()
		for {
			select {
			case <-s.term.ctx.Done():
				return
			case <-tk.C:
				requests := s.term.window.TakeTimeout()
				for _, request := range requests {
					if s.connClosed() {
						return
					}
					s.onRespond(NewResponse(request, nil, ErrResponseTimeout))
				}
				s.debug("Handled timeout requests, count: %d", len(requests))
			}
		}
	}()
}

func (s *Session) onDialed() {
	s.store.AddSession(s)
	if s.conf.OnDialed != nil {
		s.conf.OnDialed(s)
	}
}

func (s *Session) onClosed(reason string, desc string) {
	s.store.DelSession(s.id)
	if s.conf.OnClosed != nil {
		s.conf.OnClosed(s, reason, desc)
	}
}

func (s *Session) onReceive(p pdu.PDU) pdu.PDU {
	if s.conf.OnReceive != nil {
		return s.conf.OnReceive(s, p)
	}
	return nil
}

func (s *Session) onRequest(request *Request) {
	if s.conf.OnRequest != nil && request.submitter == SubmitterUser {
		s.conf.OnRequest(s, request)
	}
}

func (s *Session) onRespond(response *Response) {
	if s.conf.OnRespond != nil && response.Request.submitter == SubmitterUser {
		s.conf.OnRespond(s, response)
	}
}

func (s *Session) debug(m string, a ...any) {
	if s.slog != nil {
		s.slog.Debugf(s.formatLog(m, a...))
	}
}

func (s *Session) info(m string, a ...any) {
	if s.slog != nil {
		s.slog.Infof(s.formatLog(m, a...))
	}
}

func (s *Session) warn(m string, a ...any) {
	if s.slog != nil {
		s.slog.Warnf(s.formatLog(m, a...))
	}
}

func (s *Session) formatLog(m string, a ...any) string {
	return fmt.Sprintf("[Session@%s:%s] ", s.Id(), s.SystemId()) + fmt.Sprintf(m, a...)
}

func (s *Session) Id() string {
	return s.id
}

func (s *Session) SelfAddr() string {
	return s.conn.SelfAddr()
}

func (s *Session) PeerAddr() string {
	return s.conn.PeerAddr()
}

func (s *Session) SystemId() string {
	return s.conn.SystemId()
}

func (s *Session) BindType() pdu.BindingType {
	return s.conn.BindType()
}

func (s *Session) InitAt() time.Time {
	return s.initAt
}

func (s *Session) DialAt() time.Time {
	return s.term.dialAt
}

func (s *Session) GetContext() any {
	return s.conf.Context
}

func (s *Session) SetContext(ctx any) {
	s.conf.Context = ctx
}

func (s *Session) GetWindow() Window {
	return s.term.window
}

func (s *Session) Write(p pdu.PDU, data any) error {
	if s.connClosed() {
		return ErrConnectionClosed
	}

	s.writeToQueue(SubmitterUser, p, data)

	return nil
}

func (s *Session) Close() {
	atomic.StoreInt32(&s.closed, 1)
	s.close(CloseByExplicit, "")
}

func (s *Session) IsActive() bool {
	return s.connDialed()
}

func (s *Session) Closed() bool {
	c1 := atomic.LoadInt32(&s.closed) == 1          // 显示关闭会话
	c2 := s.conf.AttemptDial == 0 && s.connClosed() // 或连接已关闭并且没有开启重连
	return c1 || c2
}

func (s *Session) Status() string {
	if s.Closed() {
		return SessionClosed
	}
	if s.connClosed() {
		return SessionDialing
	}
	return SessionActive
}
