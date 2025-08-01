package smpp

type Authenticate func(systemId string, password string) bool
