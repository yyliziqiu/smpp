package smpp

import (
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/exp/maps"
)

type SessionGroup struct {
	conf      *SessionGroupConfig
	sessions  map[string]*Session
	keys      []string
	round     int32
	adjusting int32
	destroyed bool
	mu        sync.RWMutex
}

type SessionGroupConfig struct {
	GroupId  string                                     // 会话组 ID
	Capacity int                                        // 会话组中最大会话数量
	AutoFill bool                                       // 是否自动创建会话
	Values   any                                        // AutoFill 为 true 时，用来自定义用户数据
	Create   func(*SessionGroup, any) (*Session, error) // AutoFill 为 true 时，用来自动创建会话
	Failed   func(*SessionGroup, error)                 // AutoFill 为 true 时，自动创建会话失败时执行
}

func NewSessionGroup(conf *SessionGroupConfig) *SessionGroup {
	return &SessionGroup{
		conf:     conf,
		sessions: make(map[string]*Session),
	}
}

// Id 获取会话组 ID
func (g *SessionGroup) Id() string {
	return g.conf.GroupId
}

// Config 获取会话组配置
func (g *SessionGroup) Config() *SessionGroupConfig {
	return g.conf
}

// Len 获取会话组中的会话数量
func (g *SessionGroup) Len() int {
	return g.len()
}

func (g *SessionGroup) len() int {
	return len(g.keys)
}

// Round 轮询获取会话组中的会话
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

// Get 获取会话组中指定会话
func (g *SessionGroup) Get(sessionId string) *Session {
	g.mu.RLock()
	sess := g.sessions[sessionId]
	g.mu.RUnlock()

	return sess
}

// All 获取会话组所有会话
func (g *SessionGroup) All() []*Session {
	g.mu.RLock()
	list := maps.Values(g.sessions)
	g.mu.RUnlock()

	return list
}

// Add 向会话组中添加一个会话
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
	return len(g.keys) >= g.conf.Capacity
}

func (g *SessionGroup) add(sess *Session) {
	g.sessions[sess.Id()] = sess
	g.keys = maps.Keys(g.sessions)
	logInfo("[SessionGroup@%s] Add session, session id: %s", g.Id(), sess.Id())
}

// Del 从会话组中删除一个会话
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

// Adjust 调整会话组中的会话数量
func (g *SessionGroup) Adjust() {
	if g.destroyed {
		return
	}

	if !atomic.CompareAndSwapInt32(&g.adjusting, 0, 1) {
		return
	}

	g.mu.RLock()
	diff := g.conf.Capacity - g.len()
	g.mu.RUnlock()

	if g.conf.AutoFill {
		for i := 0; i < diff; i++ {
			g.create()
		}
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

	sess, err := g.conf.Create(g, g.conf.Values)
	if err != nil {
		if g.conf.Failed != nil {
			g.conf.Failed(g, err)
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

// Capacity 设置会话组最大会话数
func (g *SessionGroup) Capacity(n int) {
	g.mu.Lock()
	g.conf.Capacity = n
	g.mu.Unlock()
	g.Adjust()
}

// Destroy 销毁会话组，并关闭会话组中的所有会话
func (g *SessionGroup) Destroy() {
	for _, session := range g.destroy() {
		session.Close()
	}
	logInfo("[SessionGroup@%s] Destroy", g.Id())
}

func (g *SessionGroup) destroy() []*Session {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.destroyed {
		return nil
	}

	sessions := maps.Values(g.sessions)

	g.sessions = nil
	g.keys = nil
	g.destroyed = true

	return sessions
}
