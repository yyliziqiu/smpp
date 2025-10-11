package assist

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/yyliziqiu/smpp/smpp"
)

func TestSessionGroupManager(t *testing.T) {
	manager := NewSessionGroupManager(SessionGroupManagerConfig{
		AdjustInterval: 5 * time.Second,
	})

	for i := 0; i < 3; i++ {
		err := manager.Register(SessionGroupManagerNewSessionGroupConfig("group" + strconv.Itoa(i+1)))
		if err != nil {
			t.Log("Error: ", err)
		}
	}

	time.Sleep(3 * time.Second)

	sg := manager.Get("group1")

	sg.Del(sg.keys[0])

	time.Sleep(3 * time.Second)
}

func SessionGroupManagerNewSessionGroupConfig(id string) SessionGroupConfig {
	return SessionGroupConfig{
		GroupId:  id,
		Capacity: 3,
		AutoFill: true,
		Values:   "test group1",
		Create: func(group *SessionGroup, val any) (*smpp.Session, error) {
			fmt.Println("create session: ", val)
			return smpp.NewSession(smpp.NewClientConnection(_clientConnectionConfig), smpp.SessionConfig{
				EnquireLink: 30 * time.Second,
				AttemptDial: 10 * time.Second,
				OnClosed: func(sess *smpp.Session, reason string, desc string) {
					group.Del(sess.Id())
					fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
				},
			})
		},
		Failed: func(group *SessionGroup, err error) {
			fmt.Println("Error: ", err)
		},
	}
}
