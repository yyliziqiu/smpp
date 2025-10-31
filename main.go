package main

import (
	"os"

	"github.com/yyliziqiu/smpp/example"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "client" {
		example.StartClient()
	} else {
		example.StartServer()
	}
}
