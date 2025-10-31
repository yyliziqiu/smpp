package example

import (
	"sync"
	"sync/atomic"

	"golang.org/x/exp/maps"

	"github.com/yyliziqiu/smpp/smpp"
)

type SessionPool struct {
	conf      *SessionPoolConfig
	list      map[string]*smpp.Session
	keys      []string
	next      int32
	destroyed bool
	adjusting int32
	mu        sync.RWMutex
}

type SessionPoolConfig struct {
	Key    string
	Cap    int
	Create func(*SessionPool) (*smpp.Session, error)
	Failed func(*SessionPool, error)
}

func NewSessionPool(conf *SessionPoolConfig) *SessionPool {
	return &SessionPool{
		conf: conf,
		list: make(map[string]*smpp.Session),
		keys: make([]string, 0),
	}
}

// Key 获取会话组配置
func (t *SessionPool) Key() string {
	return t.conf.Key
}

// Len 获取会话组中的会话数量
func (t *SessionPool) Len() int {
	return t.len()
}

func (t *SessionPool) len() int {
	return len(t.keys)
}

// Get 获取会话组中指定会话
func (t *SessionPool) Get(sessionId string) *smpp.Session {
	t.mu.RLock()
	sess := t.list[sessionId]
	t.mu.RUnlock()

	return sess
}

// All 获取会话组所有会话
func (t *SessionPool) All() []*smpp.Session {
	t.mu.RLock()
	list := maps.Values(t.list)
	t.mu.RUnlock()

	return list
}

// Next 轮询获取会话组中的会话
func (t *SessionPool) Next() (*smpp.Session, bool) {
	var sess *smpp.Session

	t.mu.RLock()
	n := int32(t.len())
	if n > 0 {
		i := atomic.AddInt32(&t.next, 1) & 0x7FFFFFFF
		sess = t.list[t.keys[i%n]]
	}
	t.mu.RUnlock()

	return sess, n > 0
}

// Del 从会话组中删除一个会话
func (t *SessionPool) Del(sessionId string) {
	var sess *smpp.Session

	t.mu.Lock()
	if !t.destroyed {
		sess = t.del(sessionId)
	}
	t.mu.Unlock()

	if sess != nil {
		sess.Close()
		t.Adjust()
	}
}

func (t *SessionPool) del(sessionId string) *smpp.Session {
	sess, ok := t.list[sessionId]
	if ok {
		delete(t.list, sessionId)
		t.keys = maps.Keys(t.list)
	}
	return sess
}

// Destroy 销毁会话组，并关闭会话组中的所有会话
func (t *SessionPool) Destroy() {
	for _, session := range t.destroy() {
		session.Close()
	}
}

func (t *SessionPool) destroy() []*smpp.Session {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.destroyed {
		return nil
	}

	list := maps.Values(t.list)

	t.list = nil
	t.keys = nil
	t.destroyed = true

	return list
}

// Capacity 设置会话组最大会话数
func (t *SessionPool) Capacity(n int) {
	t.mu.Lock()
	t.conf.Cap = n
	t.mu.Unlock()
	t.Adjust()
}

// Adjust 调整会话组中的会话数量
func (t *SessionPool) Adjust() {
	if t.destroyed {
		return
	}

	if !atomic.CompareAndSwapInt32(&t.adjusting, 0, 1) {
		return
	}

	t.mu.RLock()
	diff := t.conf.Cap - t.len()
	t.mu.RUnlock()

	for i := 0; i < diff; i++ {
		t.create()
	}

	for i := diff; i < 0; i++ {
		sess := t.remove()
		if sess != nil {
			sess.Close()
		}
	}

	atomic.StoreInt32(&t.adjusting, 0)
}

func (t *SessionPool) create() {
	sess, err := t.conf.Create(t)
	if err != nil {
		if t.conf.Failed != nil {
			t.conf.Failed(t, err)
		}
		return
	}

	keep := false
	t.mu.Lock()
	if !t.destroyed && len(t.keys) < t.conf.Cap {
		t.list[sess.Id()] = sess
		t.keys = maps.Keys(t.list)
		keep = true
	}
	t.mu.Unlock()

	if !keep {
		sess.Close()
		return
	}
}

func (t *SessionPool) remove() *smpp.Session {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.destroyed || len(t.keys) == 0 {
		return nil
	}

	return t.del(t.keys[0])
}
