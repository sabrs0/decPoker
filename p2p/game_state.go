package p2p

import (
	"fmt"
	"sync"
	atomic "sync/atomic"

	"time"

	"github.com/sirupsen/logrus"
)

type Player struct {
	Status     GameStatus
	ListenAddr string
}

type GameState struct {
	listenAddr             string
	isDealer               bool       //sould be atomic accessable
	gameStatus             GameStatus //sould be atomic accessable
	playerLock             sync.RWMutex
	PlayersWaitingForCards int32
	playersList            []*Player
	players                map[string]*Player
	broadCastCh            chan BroadCastTo
	DecksRecieved          map[string]bool
	DecksRecievedLock      sync.RWMutex
}

func NewGameState(addr string, bch chan BroadCastTo) *GameState {
	g := &GameState{
		listenAddr:    addr,
		isDealer:      false,
		gameStatus:    GameStatusWaiting,
		players:       make(map[string]*Player),
		broadCastCh:   bch,
		DecksRecieved: make(map[string]bool),
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
		g.InitiateAndDealCards()
	}

}

func (g *GameState) GetPlayersWithStatus(s GameStatus) []string {

	players := []string{}
	for addr, player := range g.players {
		if player.Status == s {
			players = append(players, addr)
		}

	}
	return players
}

func (g *GameState) SendToPlayerWithStatus(payload any, s GameStatus) {
	players := g.GetPlayersWithStatus(s)
	g.broadCastCh <- BroadCastTo{
		Payload: payload,
		To:      players,
	}
	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"players": players,
	}).Info("sending to players ")
}

func (g *GameState) ShuffleAndEncrypt(from string, deck [][]byte) error {

	dealToPlayer := g.playersList[1]

	logrus.WithFields(logrus.Fields{
		"from":           from,
		"we":             g.listenAddr,
		"deal to player": dealToPlayer.ListenAddr,
	}).Info("recieved cards and going to shuffle")
	//TODO: Get this player out of determenistic list
	//TODO: Shuffle and Deal

	fmt.Println(dealToPlayer)
	return nil
}

func (g *GameState) InitiateAndDealCards() {
	dealToPlayer := g.playersList[0]

	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
	g.SetStatus(GameStatusShuffleAndDeal)

}

func (g *GameState) SendToPlayer(addr string, payload any) {
	g.broadCastCh <- BroadCastTo{
		Payload: payload,
		To:      []string{addr},
	}
	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"player":  addr,
	}).Info("sending to players ")
}
func (g *GameState) setDecksRecieved(from string) {
	g.DecksRecievedLock.Lock()
	g.DecksRecieved[from] = true
	g.DecksRecievedLock.Unlock()
}

func (g *GameState) DealCards() {

	//g.broadCastCh <- MessageEncDeck{Deck: [][]byte{}}
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

	player := &Player{
		ListenAddr: addr,
	}
	g.players[addr] = player
	g.playersList = append(g.playersList, player)
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
			"decks recieved":    g.DecksRecieved,
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
