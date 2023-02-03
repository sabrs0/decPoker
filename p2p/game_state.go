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

type PlayersReady struct {
	mu           sync.RWMutex
	recvStatuses map[string]bool
}

func NewPlayersReady() *PlayersReady {
	return &PlayersReady{
		recvStatuses: make(map[string]bool),
	}
}

// recvStatuses - addresses of players who is READY
func (pr *PlayersReady) addRecvStatus(from string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.recvStatuses[from] = true
}

func (pr *PlayersReady) haveRecv(from string) bool {
	_, ok := pr.recvStatuses[from]
	return ok
}

func (pr *PlayersReady) len() int {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return len(pr.recvStatuses)
}

func (pr *PlayersReady) Clear() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.recvStatuses = make(map[string]bool)

}

type Game struct {
	playersReady *PlayersReady

	playersList PlayersList
	listenAddr  string
	broadCastCh chan BroadCastTo
	//currentStatus should be atomically accessable
	currentStatus GameStatus
	currentDealer int32
}

func NewGame(addr string, bc chan BroadCastTo) *Game {
	g := &Game{
		currentStatus: GameStatusPlayerConnected,
		playersReady:  NewPlayersReady(),
		playersList:   PlayersList{},
		listenAddr:    addr,
		broadCastCh:   bc,
		currentDealer: -1,
	}

	g.playersList = append(g.playersList, g.listenAddr)
	go g.loop()
	return g
}

/*


 */
func (g *Game) ShuffleAndEncrypt(from string, deck [][]byte) error {

	//external check if we REALLY recieved enc deck from PREVIOUS player
	prevPlayerAddr := g.playersList[g.GetPrevPosOnTable()]
	if from != prevPlayerAddr {
		return fmt.Errorf("recieved encrypted deck from the wrong player %s. should be %s\n", from, prevPlayerAddr)
	}
	_, isDealer := g.getCurrentDealerAddr()
	if isDealer && from == prevPlayerAddr {
		logrus.Info("end of round")
		return nil
	}

	dealToPlayer := g.playersList[g.GetNextPosOnTable()]
	//dealToPlayer := g.GetNextReadyPlayer(g.GetOurPositionOnTable())
	logrus.WithFields(logrus.Fields{
		"from":           from,
		"we":             g.listenAddr,
		"deal to player": dealToPlayer,
	}).Info("recieved cards and going to shuffle")

	//TODO: Get this player out of determenistic list
	//TODO: Shuffle and Deal

	g.SendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayer)
	g.SetStatus(GameStatusDealing)

	return nil
}

func (g *Game) InitiateShuffleAndDeal() {
	dealToPlayerAddr := g.playersList[g.GetNextPosOnTable()]
	//NOTE: Разкомментить надо наверное
	//g.SetStatus(GameStatusDealing)
	g.SendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayerAddr)

	logrus.WithFields(logrus.Fields{
		"we": g.listenAddr,
		"to": dealToPlayerAddr,
	}).Info("dealing cards")

}

/*




 */

// TODO: Check other read and write occurances of the GameState
func (g *Game) SetStatus(s GameStatus) {
	if g.currentStatus != s {
		atomic.StoreInt32((*int32)(&g.currentStatus), (int32)(s))

		//g.SetPlayerStatus(g.listenAddr, s)
	}

}

func (g *Game) getCurrentDealerAddr() (string, bool) {
	currentDealer := g.playersList[0]

	if g.currentDealer > -1 {
		currentDealer = g.playersList[g.currentDealer]
	}

	return currentDealer, currentDealer == g.listenAddr
}

func (g *Game) setPlayerReady(from string) {
	g.playersReady.addRecvStatus(from)

	logrus.WithFields(logrus.Fields{
		"we":          g.listenAddr,
		"player from": from,
	}).Info("setting player status ready")

	//NOTE: Here need to check if round can be started
	//if we dont have enough players, the round cannot be started
	if g.playersReady.len() < 2 {
		return
	}

	//in this case the round can be started, hence the round can be started
	//FIXME:
	//g.playersReady.Clear()

	if _, areWe := g.getCurrentDealerAddr(); areWe {
		g.InitiateShuffleAndDeal()
	}

}

//NOTE: Notify other players that we are ready. Used in api.go
func (g *Game) SetReady() {
	g.playersReady.addRecvStatus(g.listenAddr)
	g.SetStatus(GameStatusPlayerReady)
	g.SendToPlayers(MessageReady{}, g.getOtherPlayers()...)
}

func (g *Game) SendToPlayers(payload any, addr ...string) {

	g.broadCastCh <- BroadCastTo{
		Payload: payload,
		To:      addr,
	}
	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"players": addr,
		"we":      g.listenAddr,
	}).Info("GameState sending to players ")
}

func (g *Game) AddPlayer(from string) {
	//if player is being added to the game. We assume
	//that he is ready to play
	g.playersList = append(g.playersList, from)
	//g.playersReady.addRecvStatus(from)
	//g.setPlayerReady(from)
	sort.Sort(g.playersList)
}

func (g *Game) loop() {
	//time.Sleep(time.Nanosecond)
	ticker := time.NewTicker(time.Second * 5)
	for x := range ticker.C {
		_ = x
		curDealer, _ := g.getCurrentDealerAddr()
		logrus.WithFields(logrus.Fields{
			"we":                g.listenAddr,
			"players connected": g.playersList,
			"status":            g.currentStatus,
			"current dealer":    curDealer,
			"players ready":     g.playersReady.recvStatuses,
		}).Info("CHECKK")
	}
}

func (g *Game) getOtherPlayers() []string {
	players := []string{}

	for _, addr := range g.playersList {
		if addr != g.listenAddr {
			players = append(players, addr)
		}
	}
	return players
}

/*







 */

func (g *Game) GetOurPositionOnTable() int {
	for i := 0; i < len(g.playersList); i++ {
		if g.playersList[i] == g.listenAddr {
			return i
		}
	}
	panic("player not in the list, this is not gonna happen")
}
func (g *Game) GetPrevPosOnTable() int {
	ourPos := g.GetOurPositionOnTable()

	if ourPos == 0 {
		return len(g.playersList) - 1
	}
	return ourPos - 1
}
func (g *Game) GetNextPosOnTable() int {
	ourPos := g.GetOurPositionOnTable()

	if ourPos == len(g.playersList)-1 {
		return 0
	}
	return ourPos + 1
}

/*func (g *Game) GetNextReadyPlayer() string {
	nextPos := g.GetOurPositionOnTable()

	if ourPos == len(g.playersList)-1 {
		return 0
	}
	return ourPos + 1
}*/

//NOTE: RECURSION
func (g *Game) GetNextReadyPlayer(pos int) string {
	nextPos := (pos + 1) % g.playersList.Len() // g.GetNextPosOnTable()
	nextPlayerAddr := g.playersList[nextPos]
	if g.playersReady.haveRecv(nextPlayerAddr) {
		return nextPlayerAddr
	}
	return g.GetNextReadyPlayer(pos + 1)
}

//for {
//	<-ticker.C
//	logrus.WithFields(logrus.Fields{
//		"we":                g.listenAddr,
//		"players connected": g.playersList,
//		"status":            g.gameStatus,
//	}).Info("CHECKK")
//}

/*
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

//func (g *GameState) AddPlayerWaitingForCards() {
//	atomic.AddInt32(&g.PlayersWaitingForCards, 1)
//}

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

func (g *Game) InitiateAndDealCards() {
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

	///f status == GameStatusWaiting {
	//	g.AddPlayerWaitingForCards()
	//}

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


*/
type PlayersList []string

func (list PlayersList) Len() int { return len(list) }

func (list PlayersList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(list[i][1:])
	portJ, _ := strconv.Atoi(list[j][1:])

	return portI < portJ
}

/*type Player struct {
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
}*/
