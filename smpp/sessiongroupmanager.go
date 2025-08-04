package smpp

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

type SessionGroupManager struct {
	config *SessionGroupManagerConfig
	groups map[string]*SessionGroup
	adjust map[string]*SessionGroup
	mu     sync.RWMutex
}

type SessionGroupManagerConfig struct {
	AdjustInterval time.Duration
}

func NewSessionGroupManager(config SessionGroupManagerConfig) *SessionGroupManager {
	m := &SessionGroupManager{
		config: &config,
		groups: make(map[string]*SessionGroup),
		adjust: make(map[string]*SessionGroup),
	}

	go m.runAdjusts()

	return m
}

func (m *SessionGroupManager) runAdjusts() {
	ticker := time.NewTicker(m.config.AdjustInterval)
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
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[conf.GroupId]; ok {
		return fmt.Errorf("group %s already exists", conf.GroupId)
	}

	group := NewSessionGroup(&conf)
	group.Adjust()

	m.groups[conf.GroupId] = group
	if conf.AutoFill {
		m.adjust[conf.GroupId] = group
	}

	return nil
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
