package p2p

import (
	"bytes"
	"io"
	"log"
	"net"
)

type Message struct {
	From    net.Addr
	Payload io.Reader
}

type Peer struct {
	conn net.Conn
}

func (p *Peer) Send(b []byte) error {
	_, err := p.conn.Write(b)
	return err
}
func (p *Peer) ReadLoop(msgCh chan *Message) {
	defer p.conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := p.conn.Read(buf)
		if err != nil {
			break
		}
		msgCh <- &Message{
			From:    p.conn.RemoteAddr(),
			Payload: bytes.NewReader(buf[:n]),
		}
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
			conn: c,
		}
		t.AddPeer <- peer
	}
}
