package smpp

import (
	"fmt"
	"testing"
)

func TestParseDlr(t *testing.T) {
	s := "id:1751354780682325918 sub:001 dlvrd:001 submit date:2507251650 done date:2507251820 stat:DELIVRD err:000 text:123"

	r, err := ParseDlr(s)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", r)

	fmt.Println(r.String())
}
