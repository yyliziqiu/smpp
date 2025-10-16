package main

import (
	"os"

	"github.com/yyliziqiu/smpp/ex"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "client" {
		ex.StartClient()
	} else {
		ex.StartServer()
	}
}
