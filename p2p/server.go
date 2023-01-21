package p2p

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
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

	transport   *TCPTransport
	peers       map[net.Addr]*Peer
	addPeer     chan *Peer
	delPeer     chan *Peer
	msgCh       chan *Message
	peerLock    sync.RWMutex
	broadcastCh chan any

	gameState *GameState
}

func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		ServerConfig: cfg,
		peers:        make(map[net.Addr]*Peer),
		addPeer:      make(chan *Peer, 10),
		delPeer:      make(chan *Peer),
		msgCh:        make(chan *Message),
		broadcastCh:  make(chan any),
	}
	s.gameState = NewGameState(s.ListenAddr, s.broadcastCh)
	//NOTE: JUST FOR TEST, DELETE THIS BLOCK LATER
	if s.ListenAddr == ":3000" {
		s.gameState.isDealer = true
	}

	tr := NewTCPTransport(s.ListenAddr)

	s.transport = tr

	tr.AddPeer = s.addPeer
	tr.DelPeer = s.delPeer

	return s

}

func (s *Server) Start() {
	go s.loop()

	//go s.gameState.loop()
	logrus.WithFields(logrus.Fields{
		"listenAddr":  s.ListenAddr,
		"variant":     s.GameVariant,
		"game status": s.gameState.gameStatus,
	}).Info("server starts listening ")
	go func() {
		for {
			select {
			case msg := <-s.broadcastCh:
				logrus.Info("BroadCasting to peers")
				if err := s.BroadCast(msg); err != nil {
					logrus.Errorf("broadcast error :", err)
				}
			}
		}
	}()
	s.transport.ListenAndAccept()
}

//server connects to other server and creates  a peer on itself and makes this peer to send handshake
func (s *Server) Connect(addr string) error {
	if s.IsInPeerList(addr) {
		return nil
	}
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

func (s *Server) Peers() []string {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()

	peers := make([]string, len(s.peers))
	it := 0
	for _, peer := range s.peers {
		peers[it] = peer.ListenAddr
		it++
	}
	return peers
}

func (s *Server) AddPeer(p *Peer) {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	//NOTE: вощможно тут надо s.peers[p.ListenAddr] = p
	s.peers[p.conn.RemoteAddr()] = p

}

func (s *Server) BroadCast(payload any) error {
	msg := NewMessage(s.ListenAddr, payload)
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	for _, peer := range s.peers {
		go func(p *Peer) {
			if err := p.Send(buf.Bytes()); err != nil {
				logrus.Errorf("broadCast to peer error %s", err)
			}
			logrus.WithFields(logrus.Fields{
				"we":   s.ListenAddr,
				"peer": p.ListenAddr,
			}).Info("broadcast")
		}(peer)
	}
	return nil
}

func (s *Server) loop() {
	//коварная ошибОчка: мы глушимся на том, что бродкастим и одновременно читаем из канала msg. И возникает ситуация, когда готов и бродкаст и msg и что- то из
	//этого теряется в итоге

	for {
		select {
		case peer := <-s.delPeer:
			logrus.WithFields(logrus.Fields{
				"addr": peer.conn.RemoteAddr(),
			}).Info("player disconnected")
			delete(s.peers, peer.conn.RemoteAddr())

		case peer := <-s.addPeer:
			if err := s.handleNewPeer(peer); err != nil {
				logrus.Errorf("handle peer error : %s", err)
				//continue
			}
		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				logrus.Errorf("handle msg error : %s", err)
			}
		}
	}
}
func (s *Server) handleNewPeer(peer *Peer) error {
	hs, err := s.handshake(peer)
	if err != nil {
		peer.conn.Close()
		delete(s.peers, peer.conn.RemoteAddr())

		return fmt.Errorf("handshake with incoming player failed %s", err.Error())

	}

	// NOTE: Thist loop must be started after the handshake
	go peer.ReadLoop(s.msgCh)

	if !peer.outbound {
		if err := s.SendHandshake(peer); err != nil {

			peer.conn.Close()
			delete(s.peers, peer.conn.RemoteAddr())
			return fmt.Errorf("failed to send handshake with peer : %s", err.Error())
		}
		go func() {
			if err := s.SendPeerList(peer); err != nil {
				logrus.Errorf("peerlist error : %s", err.Error())

			}
		}()
	}
	//TODO: Check max players and other game states

	logrus.WithFields(logrus.Fields{
		"me":           s.ListenAddr,
		"peer":         peer.conn.RemoteAddr(),
		"version":      hs.Version,
		"Game Variant": hs.GameVariant,
		"Game Status":  hs.GameStatus,
		"listenAddr":   peer.ListenAddr,
	}).Info("Recieved handshake")

	s.AddPeer(peer)
	s.gameState.AddPlayer(peer.ListenAddr, hs.GameStatus)
	return nil
}
func (s *Server) handshake(p *Peer) (*Handshake, error) {
	hs := &Handshake{}
	if err := gob.NewDecoder(p.conn).Decode(hs); err != nil {
		return nil, err
	}
	if s.GameVariant != hs.GameVariant {
		return nil, fmt.Errorf(" game variant does not match :%s", hs.GameVariant)
	}
	if s.Version != hs.Version {
		return nil, fmt.Errorf("invalid version %s", hs.Version)
	}
	p.ListenAddr = hs.ListenAddr
	return hs, nil
}

//Server makes peer to send handshake
func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		Version:     s.Version,
		GameVariant: s.GameVariant,
		GameStatus:  s.gameState.gameStatus,
		ListenAddr:  s.ListenAddr,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(hs); err != nil {
		return err
	}
	return p.Send(buf.Bytes())
}
func (s *Server) IsInPeerList(addr string) bool {
	peers := s.Peers()
	for i := 0; i < len(peers); i++ {
		if addr == peers[i] {
			return true
		}
	}
	return false
}
func (s *Server) SendPeerList(p *Peer) error {

	peerlist := MessagePeerList{
		Peers: []string{},
	}
	peers := s.Peers()
	for i := 0; i < len(peers); i++ {
		if peers[i] != p.ListenAddr {
			peerlist.Peers = append(peerlist.Peers, peers[i])
		}
	}

	fmt.Printf("%s : my peerlist => %+v\n", s.ListenAddr, peerlist.Peers)

	if len(peerlist.Peers) == 0 {
		return nil
	}

	msg := NewMessage(s.ListenAddr, peerlist)

	buffer := new(bytes.Buffer)

	if err := gob.NewEncoder(buffer).Encode(msg); err != nil {
		return err
	}
	return p.Send(buffer.Bytes())
}

func (s *Server) handleMessage(msg *Message) error {

	switch v := msg.Payload.(type) {
	case MessagePeerList:
		return s.handlePeerList(v)
	case MessageCards:
		logrus.WithFields(logrus.Fields{
			"we":           s.ListenAddr,
			"message from": msg.From,
			"cards":        v.Deck,
		}).Info("Recieved cards")
		s.gameState.SetStatus(GameStatusRecievingCards)
	default:
		fmt.Println("unknown")
	}
	return nil
}
func (s *Server) handlePeerList(l MessagePeerList) error {
	//Maybe gorountine

	logrus.WithFields(logrus.Fields{
		"we":       s.ListenAddr,
		"PeerList": l.Peers,
	}) //.Info("recieved peerlist message")
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
	gob.Register(MessageCards{})
}
