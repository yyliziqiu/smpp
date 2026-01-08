package smpp

import (
	"fmt"
	"testing"
	"time"
)

func TestParseDlr(t *testing.T) {
	strs := []string{
		"id:1081737143648 sub:000 dlvrd:000 submit date:2605111200 done date:2605111200 stat:DELIVRD err:000 text:",
		"id:1081737143648 sub:000 dlvrd:000 submit date:260511120000 done date:260511120000 stat:DELIVRD err:000 text:",
		"id:1081737143648 sub:000 dlvrd:000 submit date:1767865032 done date:1767865032 stat:DELIVRD err:000 text:",
		"id:1081737143648 sub:000 dlvrd:000 submit date:1767865032314 done date:1767865032314 stat:DELIVRD err:000 text:",
		"id:1081737143648 sub:000 dlvrd:000 submit date:176786503 done date:176786503 stat:DELIVRD err:000 text:",
	}

	for _, str := range strs {
		dlr, err := ParseDlr(str)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(dlr.Sd.Format(time.DateTime))
	}
}
