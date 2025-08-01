package smpp

import (
	"maps"
	"sync"
)

type Tracer struct {
	ts map[string]*Session
	mu sync.RWMutex
}

func NewTracer() *Tracer {
	return &Tracer{
		ts: make(map[string]*Session),
	}
}

func (t *Tracer) GetSession(id string) *Session {
	t.mu.RLock()
	sess := t.ts[id]
	t.mu.RUnlock()

	return sess
}

func (t *Tracer) GetSessions() map[string]*Session {
	t.mu.RLock()
	sess := maps.Clone(t.ts)
	t.mu.RUnlock()

	return sess
}

func (t *Tracer) CountSessions() int {
	t.mu.RLock()
	n := len(t.ts)
	t.mu.RUnlock()

	return n
}

func (t *Tracer) AddSession(sess *Session) {
	t.mu.Lock()
	t.ts[sess.Id()] = sess
	t.mu.Unlock()

	logDebug("[Tracer] Add session, id: %s, system id: %s", sess.Id(), sess.SystemId())
}

func (t *Tracer) DelSession(id string) {
	t.mu.Lock()
	sess := t.ts[id]
	delete(t.ts, id)
	t.mu.Unlock()

	systemId := ""
	if sess != nil {
		systemId = sess.SystemId()
	}

	logDebug("[Tracer] Del session, id: %s, system id: %s", id, systemId)
}
