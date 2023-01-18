package main

import (
	/*	"fmt"
		"math/rand"

	*/
	"time"
	//deck "github.com/sabrs0/decPoker/deck"
	p2p "github.com/sabrs0/decPoker/p2p"
)

func makeAndStart(addr string) *p2p.Server {
	cfg := p2p.ServerConfig{
		ListenAddr:  addr,
		Version:     "DecPoker V1.0",
		GameVariant: p2p.TexasHoldem,
	}
	server := p2p.NewServer(cfg)

	go server.Start()

	time.Sleep(time.Second)

	return server
}

func main() {

	playerA := makeAndStart(":3000")
	playerB := makeAndStart(":4000")
	playerC := makeAndStart(":5000")
	playerD := makeAndStart(":6000")

	playerB.Connect(playerA.ListenAddr)
	time.Sleep(2 * time.Millisecond)

	playerC.Connect(playerB.ListenAddr)
	time.Sleep(2 * time.Millisecond)

	playerD.Connect(playerC.ListenAddr)

	time.Sleep(2 * time.Millisecond)

	playerB.Connect(playerC.ListenAddr)
	/*	_ = playerA
		_ = playerB*/
	select {}
}
