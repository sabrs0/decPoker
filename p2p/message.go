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

type MessageReady struct {
}

func (msg MessageReady) String() string {
	return "MSG : ready"
}

type MessagePreFlop struct {
}

func (msg MessagePreFlop) String() string {
	return "MSG : PreFlop"
}

type MessagePlayerAction struct {
	// current status is the status of the player sending his game
	// it needs to be exact the same as ours
	CurrentGameStatus GameStatus
	// action is the action player willing to take
	CurrentAction PlayerAction
	//the value of the bet if any
	Value int
}
