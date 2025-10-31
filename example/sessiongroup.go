package example

import (
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/exp/maps"

	"github.com/yyliziqiu/smpp/smpp"
)

type SessionGroup struct {
	cap       int
	list      map[string]*smpp.Session
	keys      []string
	next      int32
	destroyed bool
	mu        sync.RWMutex
}

func NewSessionGroup(cap int) *SessionGroup {
	return &SessionGroup{
		cap:  cap,
		list: make(map[string]*smpp.Session),
	}
}

// Len 获取会话组中的会话数量
func (g *SessionGroup) Len() int {
	return g.len()
}

func (g *SessionGroup) len() int {
	return len(g.keys)
}

// Get 获取会话组中指定会话
func (g *SessionGroup) Get(sessionId string) (*smpp.Session, bool) {
	g.mu.RLock()
	sess, ok := g.list[sessionId]
	g.mu.RUnlock()

	return sess, ok
}

// GetAll 获取会话组所有会话
func (g *SessionGroup) GetAll() []*smpp.Session {
	g.mu.RLock()
	list := maps.Values(g.list)
	g.mu.RUnlock()

	return list
}

// Next 轮询获取会话组中的会话
func (g *SessionGroup) Next() (*smpp.Session, bool) {
	var sess *smpp.Session

	g.mu.RLock()
	n := int32(g.len())
	if n > 0 {
		i := atomic.AddInt32(&g.next, 1) & 0x7FFFFFFF
		sess = g.list[g.keys[i%n]]
	}
	g.mu.RUnlock()

	return sess, n > 0
}

// Add 向会话组中添加一个会话
func (g *SessionGroup) Add(sess *smpp.Session) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.destroyed {
		return fmt.Errorf("session group has be destroyed")
	}

	if g.len() >= g.cap {
		return fmt.Errorf("session group has be full")
	}

	g.list[sess.Id()] = sess

	g.keys = maps.Keys(g.list)

	return nil
}

// Del 从会话组中删除一个会话
func (g *SessionGroup) Del(sessionId string) {
	g.mu.Lock()
	sess := g.del(sessionId)
	g.mu.Unlock()

	if sess != nil {
		sess.Close()
	}
}

func (g *SessionGroup) del(sessionId string) *smpp.Session {
	sess, ok := g.list[sessionId]
	if ok {
		delete(g.list, sessionId)
		g.keys = maps.Keys(g.list)
	}

	return sess
}

// Destroy 销毁会话组，并关闭会话组中的所有会话
func (g *SessionGroup) Destroy() {
	g.mu.Lock()

	if g.destroyed {
		g.mu.Unlock()
		return
	}

	sessions := maps.Values(g.list)

	g.list = nil
	g.keys = nil
	g.destroyed = true

	g.mu.Unlock()

	for _, sess := range sessions {
		sess.Close()
	}
}

// SetCap 设置会话组容量
func (g *SessionGroup) SetCap(cap int) {
	g.mu.Lock()

	g.cap = cap

	if g.destroyed {
		g.mu.Unlock()
		return
	}

	var deleted []*smpp.Session
	for i := g.len() - g.cap; i > 0; i-- {
		if len(g.keys) > 0 {
			deleted = append(deleted, g.del(g.keys[0]))
		}
	}

	g.mu.Unlock()

	for _, sess := range deleted {
		sess.Close()
	}
}
