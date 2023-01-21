package p2p

import (
	"encoding/gob"
	"log"
	"net"

	"github.com/sirupsen/logrus"
)

type NetAddr string

func Network(n NetAddr) string { return "tcp" }
func String(n NetAddr) string  { return string(n) }

type Peer struct {
	conn       net.Conn
	outbound   bool
	ListenAddr string
}

//система работает таким образом, что сюда мы отправляем не абы что, а message
func (p *Peer) Send(b []byte) error {
	_, err := p.conn.Write(b)
	return err
}
func (p *Peer) ReadLoop(msgCh chan *Message) {
	defer p.conn.Close()
	for {

		msg := new(Message)
		if err := gob.NewDecoder(p.conn).Decode(msg); err != nil {
			logrus.Errorf("decode message error : %s", err)
			break
		}
		msgCh <- msg
	}
}

type TCPTransport struct {
	listenAddr string
	listener   net.Listener

	AddPeer chan *Peer
	DelPeer chan *Peer
}

func NewTCPTransport(laddr string) *TCPTransport {
	return &TCPTransport{
		listenAddr: laddr,
	}
}
func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		return err
	}
	t.listener = ln
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Fatalf("error in accept at tcp transport:", err)
			continue
		}
		peer := &Peer{
			conn:     c,
			outbound: false,
		}
		t.AddPeer <- peer
	}
}
