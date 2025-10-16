package main

import (
	"fmt"
	"os"

	"github.com/yyliziqiu/smpp/ex"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: smpp [client]")
	}

	if os.Args[1] == "client" {
		ex.StartClient()
	} else {
		ex.StartServer()
	}
}
