package smpp

import (
	"fmt"
	"testing"
	"time"
)

func TestSessionGroup(t *testing.T) {
	group := NewSessionGroup(&SessionGroupConfig{
		GroupId:  "group1",
		Capacity: 3,
		AutoFill: true,
		Create: func(group *SessionGroup) (*Session, error) {
			fmt.Println("create session")
			return SessionGroupNewSession(group)
		},
		Failed: func(group *SessionGroup, err error) {
			fmt.Println("Error: ", err)
		},
	})

	group.Adjust()

	fmt.Println("group len: ", group.len())

	if group.len() > 0 {
		group.Del(group.keys[0])
	}

	fmt.Println("group len: ", group.len())

	time.Sleep(time.Hour)
}

func SessionGroupNewSession(group *SessionGroup) (*Session, error) {
	conf := SessionConfig{
		EnquireLink: 30 * time.Second,
		AttemptDial: 10 * time.Second,
		OnClosed: func(sess *Session, reason string, desc string) {
			group.Del(sess.Id())
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	return NewSession(NewClientConnection(_clientConnectionConfig), conf)
}
