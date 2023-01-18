package p2p

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"time"

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
		addPeer:      make(chan *Peer, 10),
		delPeer:      make(chan *Peer),
		msgCh:        make(chan *Message),
		gameState:    NewGameState(),
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
		"listenAddr":  s.ListenAddr,
		"variant":     s.GameVariant,
		"game status": s.gameState.gameStatus,
	}).Info("server starts listening ")
	s.transport.ListenAndAccept()
}

//server connects to other server and creates  a peer on itself and makes this peer to send handshake
func (s *Server) Connect(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return err
	}
	peer := &Peer{
		conn:     conn,
		outbound: true,
	}

	s.addPeer <- peer
	return s.SendHandshake(peer)

}

func (s *Server) handshake(p *Peer) error {
	hs := &Handshake{}
	if err := gob.NewDecoder(p.conn).Decode(hs); err != nil {
		return err
	}
	if s.GameVariant != hs.GameVariant {
		return fmt.Errorf(" game variant does not match :%s", hs.GameVariant)
	}
	if s.Version != hs.Version {
		return fmt.Errorf("invalid version %s", hs.Version)
	}
	logrus.WithFields(logrus.Fields{
		"me":           s.ListenAddr,
		"peer":         p.conn.RemoteAddr(),
		"version":      hs.Version,
		"Game Variant": hs.GameVariant,
		"Game Status":  hs.GameStatus,
	}).Info("Recieved handshake")
	return nil
}

//Server makes peer to send handshake
func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		Version:     s.Version,
		GameVariant: s.GameVariant,
		GameStatus:  s.gameState.gameStatus,
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
			if err := s.handshake(peer); err != nil {
				logrus.Errorf("handshake with incoming player failed %s", err.Error())
				peer.conn.Close()
				continue
			}
			go peer.ReadLoop(s.msgCh)

			if !peer.outbound {
				if err := s.SendHandshake(peer); err != nil {
					logrus.Errorf("failed to send handshake with peer : %s", err.Error())
					peer.conn.Close()
					delete(s.peers, peer.conn.RemoteAddr())
				}
				if err := s.SendPeerList(peer); err != nil {
					logrus.Errorf("peerlist error : %s", err.Error())
					continue
				}
			}
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
func (s *Server) SendPeerList(p *Peer) error {

	peerlist := MessagePeerList{
		Peers: make([]string, len(s.peers)),
	}

	it := 0
	for addr := range s.peers {
		peerlist.Peers[it] = addr.String()
		it++
	}

	msg := NewMessage(s.ListenAddr, peerlist)

	buffer := new(bytes.Buffer)

	if err := gob.NewEncoder(buffer).Encode(msg); err != nil {
		return err
	}
	return p.Send(buffer.Bytes())
}
func (s *Server) handleMessage(msg *Message) error {
	logrus.WithFields(logrus.Fields{
		"From": msg.From,
	}).Info("recieved message")

	switch v := msg.Payload.(type) {
	case MessagePeerList:
		return s.handlePeerList(v)
	}
	return nil
}
func (s *Server) handlePeerList(l MessagePeerList) error {
	//Maybe gorountine
	fmt.Printf("peerlist => %+v\n", l)
	for i := 0; i < len(l.Peers); i++ {
		if err := s.Connect(l.Peers[i]); err != nil {
			logrus.Errorf("failed to dial peer %s : %s", l.Peers[i], err)
			continue
		}
	}
	return nil
}
func init() {
	gob.Register(MessagePeerList{})
}
