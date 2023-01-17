package main

import (
	/*	"fmt"
		"math/rand"
	*/"time"

	//deck "github.com/sabrs0/decPoker/deck"
	p2p "github.com/sabrs0/decPoker/p2p"
)

func main() {
	cfg := p2p.ServerConfig{
		ListenAddr:  ":3000",
		Version:     "DecPoker V1.0\n",
		GameVariant: p2p.TexasHoldem,
	}
	server := p2p.NewServer(cfg)
	go server.Start()

	remoteCfg := p2p.ServerConfig{
		ListenAddr: ":4000",
		Version:    "DecPoker V1.0\n",
	}
	remoteServer := p2p.NewServer(remoteCfg)

	go remoteServer.Start()
	time.Sleep(time.Second)
	remoteServer.Connect(":3000")

	select {}
}
