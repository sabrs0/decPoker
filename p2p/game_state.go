package p2p

type GameStatus uint32

func (g GameStatus) String() string {
	switch g {
	case GameStatusReady:
		return "READY"
	case GameStatusDealing:
		return "DEALING"
	case GameStatusPreFlop:
		return "PRE FLOP"
	case GameStatusFlop:
		return "FLOP"
	case GameStatusTurn:
		return "TURN"
	case GameStatusRiver:
		return "RIVER"
	default:
		return "unknown"
	}
}

const (
	GameStatusReady GameStatus = iota
	GameStatusDealing
	GameStatusPreFlop
	GameStatusFlop
	GameStatusTurn
	GameStatusRiver
)

type GameState struct {
	isDealer   bool       //sould be atomic accessable
	gameStatus GameStatus //sould be atomic accessable
}

func NewGameState() *GameState {
	return &GameState{}
}
func (g *GameState) loop() {

}
