package p2p

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	atomic "sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type Player struct {
	Status     GameStatus
	ListenAddr string
}

func (p *Player) String() string {
	return fmt.Sprintf("%s => %s", p.ListenAddr, p.Status)
}

type PlayerList []*Player

func (list PlayerList) Len() int { return len(list) }

func (list PlayerList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list PlayerList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(list[i].ListenAddr[1:])
	portJ, _ := strconv.Atoi(list[j].ListenAddr[1:])

	return portI < portJ
}

type GameState struct {
	listenAddr string
	isDealer   bool       //sould be atomic accessable
	gameStatus GameStatus //sould be atomic accessable
	playerLock sync.RWMutex
	//PlayersWaitingForCards int32
	playersList PlayerList
	players     map[string]*Player
	broadCastCh chan BroadCastTo
}

func NewGameState(addr string, bch chan BroadCastTo) *GameState {
	g := &GameState{
		listenAddr:  addr,
		isDealer:    false,
		gameStatus:  GameStatusWaiting,
		players:     make(map[string]*Player),
		broadCastCh: bch,
	}
	g.AddPlayer(addr, GameStatusWaiting)

	go g.loop()
	return g

}

/*func (g *GameState) AddPlayerWaitingForCards() {
	atomic.AddInt32(&g.PlayersWaitingForCards, 1)
}*/

func (g *GameState) CheckNeedDealCards() {
	//playersWaiting := atomic.LoadInt32(&g.PlayersWaitingForCards)
	playersWaiting := g.playersWaitingForCards()
	if playersWaiting == len(g.players) &&
		g.isDealer && g.gameStatus == GameStatusWaiting {

		logrus.WithFields(logrus.Fields{
			"addr": g.listenAddr,
		}).Info("Need to deal Cards")

		g.InitiateAndDealCards()

	}

}

func (g *GameState) playersWaitingForCards() int {
	totalPlayers := 0
	for i := 0; i < len(g.playersList); i++ {
		if g.playersList[i].Status == GameStatusWaiting {
			totalPlayers++
		}
	}
	return totalPlayers
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

func (g *GameState) GetOurPositionOnTable() int {
	for i := 0; i < len(g.playersList); i++ {
		if g.playersList[i].ListenAddr == g.listenAddr {
			return i
		}
	}
	panic("player not in the list, this is not gonna happen")
}
func (g *GameState) GetPrevPosOnTable() int {
	ourPos := g.GetOurPositionOnTable()

	if ourPos == 0 {
		return len(g.playersList) - 1
	}
	return ourPos - 1
}
func (g *GameState) GetNextPosOnTable() int {
	ourPos := g.GetOurPositionOnTable()

	if ourPos == len(g.playersList)-1 {
		return 0
	}
	return ourPos + 1
}

func (g *GameState) ShuffleAndEncrypt(from string, deck [][]byte) error {
	g.SetPlayerStatus(from, GameStatusShuffleAndDeal)

	prevPlayer := g.playersList[g.GetPrevPosOnTable()]
	if g.isDealer && from == prevPlayer.ListenAddr {
		logrus.Info("end of round")
		return nil
	}

	dealToPlayer := g.playersList[g.GetNextPosOnTable()]
	logrus.WithFields(logrus.Fields{
		"from":           from,
		"we":             g.listenAddr,
		"deal to player": dealToPlayer.ListenAddr,
	}).Info("recieved cards and going to shuffle")

	//TODO: Get this player out of determenistic list
	//TODO: Shuffle and Deal

	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
	g.SetStatus(GameStatusShuffleAndDeal)

	fmt.Printf("%s => Setting my own status %s\n", g.listenAddr, GameStatusShuffleAndDeal)

	return nil
}

func (g *GameState) InitiateAndDealCards() {
	dealToPlayer := g.playersList[g.GetNextPosOnTable()]

	g.SetStatus(GameStatusShuffleAndDeal)
	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})

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

func (g *GameState) SetPlayerStatus(addr string, status GameStatus) {

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

		g.SetPlayerStatus(g.listenAddr, s)
	}

}
func (g *GameState) AddPlayer(addr string, status GameStatus) {

	g.playerLock.Lock()
	defer g.playerLock.Unlock()

	/*if status == GameStatusWaiting {
		g.AddPlayerWaitingForCards()
	}*/

	player := &Player{
		ListenAddr: addr,
	}
	g.players[addr] = player
	g.playersList = append(g.playersList, player)
	sort.Sort(g.playersList)

	//Set the player's status also when we add ther player

	g.SetPlayerStatus(addr, status)
	logrus.WithFields(logrus.Fields{
		"addr":   addr,
		"status": status,
		"we":     g.listenAddr,
	}).Info("New PLayer Joined")
}

func (g *GameState) loop() {
	//time.Sleep(time.Nanosecond)
	ticker := time.NewTicker(time.Second * 5)
	for x := range ticker.C {
		_ = x
		logrus.WithFields(logrus.Fields{
			"we":                g.listenAddr,
			"players connected": g.playersList,
			"status":            g.gameStatus,
		}).Info("CHECKK")
	}
	/*for {
		<-ticker.C
		logrus.WithFields(logrus.Fields{
			"we":                g.listenAddr,
			"players connected": g.playersList,
			"status":            g.gameStatus,
		}).Info("CHECKK")
	}*/
}
