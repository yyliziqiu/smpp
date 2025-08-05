package smpp

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

type SessionGroupManager struct {
	conf   *SessionGroupManagerConfig
	groups map[string]*SessionGroup
	adjust map[string]*SessionGroup
	ch     chan *SessionGroup
	mu     sync.RWMutex
}

type SessionGroupManagerConfig struct {
	AdjustInterval time.Duration
}

func NewSessionGroupManager(conf SessionGroupManagerConfig) *SessionGroupManager {
	m := &SessionGroupManager{
		conf:   &conf,
		groups: make(map[string]*SessionGroup),
		adjust: make(map[string]*SessionGroup),
	}

	go m.runAdjusts()

	return m
}

func (m *SessionGroupManager) runAdjusts() {
	ticker := time.NewTicker(m.conf.AdjustInterval)
	defer ticker.Stop()
	for {
		<-ticker.C

		m.mu.Lock()
		adjust := maps.Values(m.adjust)
		m.mu.Unlock()

		for _, group := range adjust {
			group.Adjust()
		}
	}
}

func (m *SessionGroupManager) Register(conf SessionGroupConfig) error {
	group, err := m.register(conf)
	if err != nil {
		return err
	}

	go func() {
		group.Adjust()
	}()

	return nil
}

func (m *SessionGroupManager) register(conf SessionGroupConfig) (*SessionGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[conf.GroupId]; ok {
		return nil, fmt.Errorf("group %s already exists", conf.GroupId)
	}

	group := NewSessionGroup(&conf)

	m.groups[conf.GroupId] = group
	if conf.AutoFill {
		m.adjust[conf.GroupId] = group
	}

	return group, nil
}

func (m *SessionGroupManager) Unregister(groupId string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.groups[groupId]
	if !ok {
		return
	}

	delete(m.groups, groupId)
	delete(m.adjust, groupId)

	group.Destroy()
}

func (m *SessionGroupManager) Get(groupId string) *SessionGroup {
	m.mu.RLock()
	group := m.groups[groupId]
	m.mu.RUnlock()

	return group
}
