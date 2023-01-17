package p2p

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"

	logrus "github.com/sirupsen/logrus"
)

type GameVariant uint8

func (gv GameVariant) String() string {
	switch gv {
	case TexasHoldem:
		return "TexasHoldem"
	case Other:
		return "Other"
	default:
		return "unknown"
	}
}

const (
	TexasHoldem GameVariant = iota
	Other
)

type ServerConfig struct {
	Version     string
	ListenAddr  string
	GameVariant GameVariant
}

type Server struct {
	ServerConfig

	transport *TCPTransport
	peers     map[net.Addr]*Peer
	addPeer   chan *Peer
	delPeer   chan *Peer
	msgCh     chan *Message

	gameState *GameState
}

func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		ServerConfig: cfg,
		peers:        make(map[net.Addr]*Peer),
		addPeer:      make(chan *Peer),
		delPeer:      make(chan *Peer),
		msgCh:        make(chan *Message),
		gameState : NewGameState()
	}

	tr := NewTCPTransport(s.ListenAddr)

	s.transport = tr

	tr.AddPeer = s.addPeer
	tr.DelPeer = s.delPeer

	return s

}

func (s *Server) Start() {
	go s.loop()

	logrus.WithFields(logrus.Fields{
		"listenAddr": s.ListenAddr,
		"variant":    s.GameVariant,
	}).Info("server starts listening ")
	s.transport.ListenAndAccept()
}
func (s *Server) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	peer := &Peer{
		conn: conn,
	}

	s.addPeer <- peer
	return peer.Send([]byte(s.Version))

}

type Handshake struct {
	Version     string
	GameVariant GameVariant
}

func (s *Server) handshake(p *Peer) error {
	hs := &Handshake{}
	if err := gob.NewDecoder(p.conn).Decode(hs); err != nil {
		return err
	}
	if s.GameVariant != hs.GameVariant {
		return fmt.Errorf("invalid game variant %s", hs.GameVariant)
	}
	if s.Version != hs.Version {
		return fmt.Errorf("invalid version %s", hs.Version)
	}
	logrus.WithFields(logrus.Fields{
		"peer":         p.conn.RemoteAddr(),
		"version":      hs.Version,
		"Game Variant": hs.GameVariant,
	}).Info("Recieved handshake")
	return nil
}

func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		Version:     s.Version,
		GameVariant: s.GameVariant,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(hs); err != nil {
		return err
	}
	return p.Send(buf.Bytes())
}

func (s *Server) loop() {
	for {
		select {
		case peer := <-s.delPeer:
			logrus.WithFields(logrus.Fields{
				"addr": peer.conn.RemoteAddr(),
			}).Info("player disconnected")
			delete(s.peers, peer.conn.RemoteAddr())

		case peer := <-s.addPeer:
			s.SendHandshake(peer)
			if err := s.handshake(peer); err != nil {
				logrus.Errorf("handshake with incoming player failed %s", err.Error())
				continue
			}
			go peer.ReadLoop(s.msgCh)

			//TODO: Check max players and other game states

			logrus.WithFields(logrus.Fields{
				"addr": peer.conn.RemoteAddr(),
			}).Info("handshake successful: new player connected")
			s.peers[peer.conn.RemoteAddr()] = peer

		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				panic(err)
			}
		}
	}
}
func (s *Server) handleMessage(msg *Message) error {
	fmt.Printf("%+v", msg)
	return nil
}
