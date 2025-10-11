package assit

import (
	"fmt"
	"testing"
	"time"

	"github.com/linxGnu/gosmpp/pdu"

	"github.com/yyliziqiu/smpp/smpp"
)

var _clientConnectionConfig = smpp.ClientConnectionConfig{
	Smsc:     "127.0.0.1:10088",
	SystemId: "user1",
	Password: "user1",
	BindType: pdu.Transceiver,
}

func TestSessionGroup(t *testing.T) {
	group := NewSessionGroup(&SessionGroupConfig{
		GroupId:  "group1",
		Capacity: 3,
		AutoFill: true,
		Values:   "test group1",
		Create: func(group *SessionGroup, val any) (*smpp.Session, error) {
			fmt.Println("create session: ", val)
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

	time.Sleep(3 * time.Second)
	group.Destroy()
	fmt.Println("group len: ", group.len())

	time.Sleep(3 * time.Second)
}

func SessionGroupNewSession(group *SessionGroup) (*smpp.Session, error) {
	conf := smpp.SessionConfig{
		EnquireLink: 30 * time.Second,
		AttemptDial: 10 * time.Second,
		OnClosed: func(sess *smpp.Session, reason string, desc string) {
			group.Del(sess.Id())
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	return smpp.NewSession(smpp.NewClientConnection(_clientConnectionConfig), conf)
}

func TestSessionGroup2(t *testing.T) {
	group := NewSessionGroup(&SessionGroupConfig{
		GroupId:  "group2",
		Capacity: 3,
	})

	for i := 0; i < 5; i++ {
		sess, err := SessionGroupNewSession(group)
		if err != nil {
			t.Log("Error: ", err)
			continue
		}
		err = group.Add(sess)
		if err != nil {
			t.Log("Error: ", err)
		}
	}

	time.Sleep(3 * time.Second)
	if group.len() > 0 {
		group.Del(group.keys[0])
	}
	fmt.Println("group len: ", group.len())

	time.Sleep(3 * time.Second)
	group.Destroy()
	fmt.Println("group len: ", group.len())

	time.Sleep(3 * time.Second)
}
