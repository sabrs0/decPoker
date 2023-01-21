package main

import (
	/*
		"math/rand"

	*/
	"fmt"
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

	time.Sleep(time.Millisecond * 200)

	return server
}

func main() {

	playerA := makeAndStart(":3000")
	playerB := makeAndStart(":4000")
	playerC := makeAndStart(":5000")
	/*playerD := makeAndStart(":6000")
	playerE := makeAndStart(":7000")*/

	time.Sleep(time.Millisecond * 200)
	playerB.Connect(playerA.ListenAddr)

	time.Sleep(time.Millisecond * 200)
	playerC.Connect(playerB.ListenAddr)

	/*time.Sleep(time.Millisecond * 200)
	playerD.Connect(playerC.ListenAddr)

	time.Sleep(time.Millisecond * 200)
	playerE.Connect(playerD.ListenAddr)*/

	/*time.Sleep(time.Second * 5)
	CheckPeerList(playerA)
	CheckPeerList(playerB)
	CheckPeerList(playerC)
	CheckPeerList(playerD)
	CheckPeerList(playerE)*/
	/*	time.Sleep(2 * time.Millisecond)

		playerB.Connect(playerC.ListenAddr)*/
	/*	_ = playerA
		_ = playerB*/
	select {}
}
func CheckPeerList(s *p2p.Server) {
	fmt.Println("\nchecking ", s.ListenAddr)
	peers := s.Peers()
	for _, peer := range peers {
		fmt.Println("\t", peer)
	}
}
