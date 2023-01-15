package server

import (
	"fmt"
	"net"
	"sync"
)

type Peer struct {
	conn net.Conn
}

type ServerConfig struct {
	ListenAddr string
}

type Server struct {
	ServerConfig

	listener net.Listener
	mu       sync.RWMutex
	peers    map[net.Addr]*Peer
	addPeer  chan *Peer
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		ServerConfig: cfg,
		peers:        make(map[net.Addr]*Peer),
		addPeer:      make(chan *Peer),
	}
}

func (s *Server) Listen() error {
	lstnr, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.listener = lstnr
	return nil
}

func (s *Server) Start() {
	go s.loop()

	if err := s.Listen(); err != nil {
		panic(err)
	}
	s.acceptLoop()
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			panic(err)
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		fmt.Println(string(buf[:n]))
	}
}

func (s *Server) loop() {
	for {
		select {
		case peer := <-s.addPeer:

			s.peers[peer.conn.RemoteAddr()] = peer
			fmt.Println("New Player connected ", peer.conn.RemoteAddr())
		}
	}
}
