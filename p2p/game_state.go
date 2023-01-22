package p2p

import (
	//"fmt"
	"sync"
	atomic "sync/atomic"

	"time"

	deck "github.com/sabrs0/decPoker/deck"
	"github.com/sirupsen/logrus"
)

type GameStatus int32

func (g GameStatus) String() string {
	switch g {
	case GameStatusWaiting:
		return "WAITING FOR CARDS"
	case GameStatusRecievingCards:
		return "RECIEVING CARDS"
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
	GameStatusWaiting GameStatus = iota
	GameStatusRecievingCards
	GameStatusDealing
	GameStatusPreFlop
	GameStatusFlop
	GameStatusTurn
	GameStatusRiver
)

type Player struct {
	Status GameStatus
}

type GameState struct {
	listenAddr             string
	isDealer               bool       //sould be atomic accessable
	gameStatus             GameStatus //sould be atomic accessable
	playerLock             sync.RWMutex
	PlayersWaitingForCards int32
	players                map[string]*Player
	broadCastCh            chan any
}

func NewGameState(addr string, bch chan any) *GameState {
	g := &GameState{
		listenAddr:  addr,
		isDealer:    false,
		gameStatus:  GameStatusWaiting,
		players:     make(map[string]*Player),
		broadCastCh: bch,
	}
	go g.loop()
	return g

}
func (g *GameState) AddPlayerWaitingForCards() {
	atomic.AddInt32(&g.PlayersWaitingForCards, 1)
}

func (g *GameState) CheckNeedDealCards() {
	playersWaiting := atomic.LoadInt32(&g.PlayersWaitingForCards)
	if playersWaiting == int32(len(g.players)) &&
		g.isDealer && g.gameStatus == GameStatusWaiting {
		logrus.WithFields(logrus.Fields{
			"addr": g.listenAddr,
		}).Info("Need to deal Cards")
		g.DealCards()
	}

}
func (g *GameState) DealCards() {

	g.broadCastCh <- MessageCards{Deck: deck.New()}
}
func (g *GameState) SetPlayerStatus(addr string, status GameStatus) {
	/*g.playerLock.Lock()
	defer g.playerLock.Unlock()*/

	player, ok := g.players[addr]
	if !ok {
		panic("player could not be found, although it should exit")
	}
	player.Status = status
	g.CheckNeedDealCards()
}

// TODO: Check other read and write occurances of the GameState
func (g *GameState) SetStatus(s GameStatus) {
	if g.gameStatus != s {
		atomic.StoreInt32((*int32)(&g.gameStatus), (int32)(s))
	}

}
func (g *GameState) AddPlayer(addr string, status GameStatus) {
	g.playerLock.Lock()
	defer g.playerLock.Unlock()

	if status == GameStatusWaiting {
		g.AddPlayerWaitingForCards()
	}

	g.players[addr] = new(Player)
	//g.CheckNeedDealCards()
	//Set the player's status also when we add ther player
	g.SetPlayerStatus(addr, status)
	logrus.WithFields(logrus.Fields{
		"addr":   addr,
		"status": status,
	}).Info("New PLayer Joined")
}
func (g *GameState) LenPlayersConnectedWithLock() int {
	g.playerLock.RLock()
	defer g.playerLock.RUnlock()

	return len(g.players)
}

func (g *GameState) loop() {
	//time.Sleep(time.Nanosecond)
	ticker := time.NewTicker(time.Second * 5)
	for x := range ticker.C {
		_ = x
		logrus.WithFields(logrus.Fields{
			"we":                g.listenAddr,
			"players connected": g.LenPlayersConnectedWithLock(),
			"status":            g.gameStatus,
		}).Info("CHECKK")
	}
	/*for {
		select {
		case <-ticker.C:
			logrus.WithFields(logrus.Fields{
				"players connected": g.LenPlayersConnectedWithLock(),
				"status":            g.gameStatus,
			}).Info("CHECKK")
			//default:
		}
	}*/
}
