package smpp

import (
	"maps"
	"sync"
)

type SessionStore struct {
	ts map[string]*Session
	mu sync.RWMutex
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		ts: make(map[string]*Session),
	}
}

func (t *SessionStore) GetSession(id string) *Session {
	t.mu.RLock()
	sess := t.ts[id]
	t.mu.RUnlock()

	return sess
}

func (t *SessionStore) GetSessions() map[string]*Session {
	t.mu.RLock()
	sess := maps.Clone(t.ts)
	t.mu.RUnlock()

	return sess
}

func (t *SessionStore) CountSessions() int {
	t.mu.RLock()
	n := len(t.ts)
	t.mu.RUnlock()

	return n
}

func (t *SessionStore) AddSession(sess *Session) {
	t.mu.Lock()
	t.ts[sess.Id()] = sess
	t.mu.Unlock()

	LogDebug("[SessionStore] Add session, id: %s, system id: %s", sess.Id(), sess.SystemId())
}

func (t *SessionStore) DelSession(id string) {
	t.mu.Lock()
	sess := t.ts[id]
	delete(t.ts, id)
	t.mu.Unlock()

	systemId := ""
	if sess != nil {
		systemId = sess.SystemId()
	}

	LogDebug("[SessionStore] Del session, id: %s, system id: %s", id, systemId)
}
