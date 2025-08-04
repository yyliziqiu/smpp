package smpp

import (
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/exp/maps"
)

type SessionGroup struct {
	config    *SessionGroupConfig
	sessions  map[string]*Session
	keys      []string
	round     int32
	adjusting int32
	destroyed bool
	mu        sync.RWMutex
}

type SessionGroupConfig struct {
	GroupId  string                                // 会话组 ID
	Capacity int                                   // 会话组大小
	AutoFill bool                                  // 是否自动创建会话
	Create   func(*SessionGroup) (*Session, error) // AutoFill 为 true 时，用来自动创建会话
	Failed   func(*SessionGroup, error)            // 自动创建会话失败时执行
}

func NewSessionGroup(config *SessionGroupConfig) *SessionGroup {
	return &SessionGroup{
		config:   config,
		sessions: make(map[string]*Session),
	}
}

func (g *SessionGroup) Id() string {
	return g.config.GroupId
}

func (g *SessionGroup) Round() *Session {
	var sess *Session

	g.mu.RLock()
	n := int32(g.len())
	if n > 0 {
		i := atomic.AddInt32(&g.round, 1) & 0x7FFFFFFF
		sess = g.sessions[g.keys[i%n]]
	}
	g.mu.RUnlock()

	return sess
}

func (g *SessionGroup) len() int {
	return len(g.keys)
}

func (g *SessionGroup) Get(sessionId string) *Session {
	g.mu.RLock()
	sess := g.sessions[sessionId]
	g.mu.RUnlock()

	return sess
}

func (g *SessionGroup) Add(sess *Session) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.destroyed {
		return fmt.Errorf("session group has be destroyed")
	}

	if g.full() {
		return fmt.Errorf("session group has be full")
	}

	g.add(sess)

	return nil
}

func (g *SessionGroup) full() bool {
	return len(g.keys) >= g.config.Capacity
}

func (g *SessionGroup) add(sess *Session) {
	g.sessions[sess.Id()] = sess
	g.keys = maps.Keys(g.sessions)
	logInfo("[SessionGroup@%s] Add session, session id: %s", g.Id(), sess.Id())
}

func (g *SessionGroup) Del(sessionId string) {
	var sess *Session

	g.mu.Lock()
	if !g.destroyed {
		sess = g.del(sessionId)
	}
	g.mu.Unlock()

	if sess != nil {
		sess.Close()
		g.Adjust()
	}
}

func (g *SessionGroup) del(sessionId string) *Session {
	sess, ok := g.sessions[sessionId]
	if ok {
		delete(g.sessions, sessionId)
		g.keys = maps.Keys(g.sessions)
		logInfo("[SessionGroup@%s] Del session, session id: %s", g.Id(), sess.Id())
	} else {
		logDebug("[SessionGroup@%s] Del not exit session, session id: %s", g.Id(), sessionId)
	}
	return sess
}

func (g *SessionGroup) Adjust() {
	if !g.config.AutoFill || g.destroyed {
		return
	}

	if !atomic.CompareAndSwapInt32(&g.adjusting, 0, 1) {
		return
	}

	g.mu.RLock()
	diff := g.config.Capacity - g.len()
	g.mu.RUnlock()

	for i := 0; i < diff; i++ {
		g.create()
	}

	for i := diff; i < 0; i++ {
		sess := g.remove()
		if sess != nil {
			sess.Close()
		}
	}

	atomic.StoreInt32(&g.adjusting, 0)
}

func (g *SessionGroup) create() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.full() || g.destroyed {
		return
	}

	sess, err := g.config.Create(g)
	if err != nil {
		failed := g.config.Failed
		if failed != nil {
			failed(g, err)
		}
		logWarn("[SessionGroup@%s] Create session failed, error: %v", g.Id(), err)
		return
	}

	logInfo("[SessionGroup@%s] Create session, session id: %s", g.Id(), sess.Id())

	g.add(sess)
}

func (g *SessionGroup) remove() *Session {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.empty() || g.destroyed {
		return nil
	}

	return g.del(g.keys[0])
}

func (g *SessionGroup) empty() bool {
	return len(g.keys) == 0
}

func (g *SessionGroup) Capacity(n int) {
	g.mu.Lock()
	g.config.Capacity = n
	g.mu.Unlock()

	g.Adjust()
}

func (g *SessionGroup) Destroy() {
	g.mu.Lock()
	sessions := g.destroy()
	g.mu.Unlock()

	logInfo("[SessionGroup@%s] Destroy", g.Id())

	for _, session := range sessions {
		session.Close()
	}
}

func (g *SessionGroup) destroy() []*Session {
	if g.destroyed {
		return nil
	}

	sessions := maps.Values(g.sessions)

	g.sessions = nil
	g.keys = nil
	g.destroyed = true

	return sessions
}
