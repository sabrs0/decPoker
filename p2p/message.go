package p2p

type Message struct {
	Payload any
	From    string
}
type BroadCastTo struct {
	To      []string
	Payload any
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

type MessageEncDeck struct {
	Deck [][]byte
}
