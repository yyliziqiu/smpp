package smpp

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/linxGnu/gosmpp/pdu"
	"github.com/yyliziqiu/slib/suid"
)

func newTestSubmitSM() *pdu.SubmitSM {
	p := pdu.NewSubmitSM().(*pdu.SubmitSM)
	p.SourceAddr = Address(5, 0, "matrix")
	p.DestAddr = Address(1, 1, "6281339900520")
	p.Message = Message("68526b7e01614899")
	p.RegisteredDelivery = 1
	return p
}

func newTestDeliverSM() *pdu.DeliverSM {
	dlr := Dlr{
		Id:    suid.Get(),
		Sub:   "001",
		Dlvrd: "001",
		SDate: time.Now(),
		DDate: time.Now(),
		Stat:  "DELIVRD",
		Err:   "000",
		Text:  "success",
	}
	return dlr.Pdu("6281339900520", "matrix")
}

func logTest(tag string, systemId string, p pdu.PDU) {
	if p != nil {
		bs, _ := json.MarshalIndent(p, "", "  ")
		fmt.Printf("[%s:%s:%T] %s\n\n", tag, systemId, p, string(bs))
	}
}

func printMemory(tag string, gc bool) {
	if gc {
		runtime.GC()
		time.Sleep(time.Second)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Printf("[memory:%s] alloc: %d KB\n", tag, memStats.Alloc/1024)
	// fmt.Printf("[memory:%s] alloc: %d KB, sys: %d KB\n", tag, memStats.Alloc/1024, memStats.Sys/1024)
}
