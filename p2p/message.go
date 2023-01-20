package p2p

type Message struct {
	Payload any
	From    string
}

func NewMessage(from string, Payload any) *Message {
	return &Message{
		Payload: Payload,
		From:    from,
	}
}

type Handshake struct {
	Version     string
	GameVariant GameVariant
	GameStatus  GameStatus
	ListenAddr  string
}

type MessagePeerList struct {
	Peers []string
}
