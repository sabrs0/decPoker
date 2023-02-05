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

type PlayerAction byte

const (
	PlayerActionFold PlayerAction = iota + 1
	PlayerActionCheck
	PlayerActionBet
)

/*










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

/*










 */

// TODO: may be use player list instead
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
	//currentDealer should be atomically accessable
	currentDealer int32

	//currentPlayerTurn should be atomically accessable
	currentPlayerTurn int32
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

func (g *Game) Fold() {
	g.SetStatus(GameStatusFolded)
	g.SendToPlayers(MessagePlayerAction{
		CurrentAction:     PlayerActionFold,
		CurrentGameStatus: g.currentStatus,
	}, g.getOtherPlayers()...)
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
		g.SetStatus(GameStatusPreFlop)
		g.SendToPlayers(MessagePreFlop{}, g.getOtherPlayers()...)
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
			"action":            PlayerActionFold,
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
