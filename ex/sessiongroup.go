package ex

import (
	"fmt"
	"time"

	"github.com/linxGnu/gosmpp/pdu"

	"github.com/yyliziqiu/smpp/assit"
	"github.com/yyliziqiu/smpp/smpp"
)

func SessionGroupExample() {
	group := assit.NewSessionGroup(&assit.SessionGroupConfig{
		GroupId:  "group1",
		Capacity: 3,
		AutoFill: true,
		Values:   "test group1",
		Create: func(group *assit.SessionGroup, val any) (*smpp.Session, error) {
			fmt.Println("create session: ", val)
			return newSessionForGroup(group)
		},
		Failed: func(group *assit.SessionGroup, err error) {
			fmt.Println("Error: ", err)
		},
	})

	sess, _ := newSessionForGroup(group)
	err := group.Add(sess)
	if err != nil {
		sess.Close()
	}

	group.Del("session id")

	group.Adjust()

	group.Destroy()
}

func newSessionForGroup(group *assit.SessionGroup) (*smpp.Session, error) {
	conn := smpp.NewClientConnection(smpp.ClientConnectionConfig{
		Smsc:     "127.0.0.1:10088",
		SystemId: "user1",
		Password: "user1",
		BindType: pdu.Transceiver,
	})

	conf := smpp.SessionConfig{
		EnquireLink: 30 * time.Second,
		AttemptDial: 10 * time.Second,
		OnClosed: func(sess *smpp.Session, reason string, desc string) {
			group.Del(sess.Id())
			fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
		},
	}

	return smpp.NewSession(conn, conf)
}

func SessionGroupManagerExample() {
	manager := assit.NewSessionGroupManager(assit.SessionGroupManagerConfig{
		AdjustInterval: 5 * time.Second,
	})

	err := manager.Register(newSessionGroupConfigForManager("group1"))
	if err != nil {
		panic(err)
	}

	time.Sleep(3 * time.Second)

	sg := manager.Get("group1")

	sg.Del("session id")

	manager.Unregister("group1")

	time.Sleep(3 * time.Second)
}

func newSessionGroupConfigForManager(id string) assit.SessionGroupConfig {
	return assit.SessionGroupConfig{
		GroupId:  id,
		Capacity: 3,
		AutoFill: true,
		Values:   "test group1",
		Create: func(group *assit.SessionGroup, val any) (*smpp.Session, error) {
			fmt.Println("create session: ", val)
			conn := smpp.NewClientConnection(smpp.ClientConnectionConfig{
				Smsc:     "127.0.0.1:10088",
				SystemId: "user1",
				Password: "user1",
				BindType: pdu.Transceiver,
			})
			return smpp.NewSession(conn, smpp.SessionConfig{
				EnquireLink: 30 * time.Second,
				AttemptDial: 10 * time.Second,
				OnClosed: func(sess *smpp.Session, reason string, desc string) {
					group.Del(sess.Id())
					fmt.Printf("[Closed] system id: %s, reason: %s, desc: %s\n", sess.SystemId(), reason, desc)
				},
			})
		},
		Failed: func(group *assit.SessionGroup, err error) {
			fmt.Println("Error: ", err)
		},
	}
}
