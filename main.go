package main

import (
	/*	"fmt"
		"math/rand"
		"time"
	*/
	//deck "github.com/sabrs0/decPoker/deck"
	p2p "github.com/sabrs0/decPoker/p2p"
)

func main() {
	cfg := p2p.ServerConfig{
		ListenAddr: ":3000",
	}
	server := p2p.NewServer(cfg)
	server.Start()
}
