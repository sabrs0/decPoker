package p2p

import (
	"encoding/json"
	"net/http"

	mux "github.com/gorilla/mux"
)

type apiFunc func(w http.ResponseWriter, r *http.Request) error

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			//почему мапа ? Может джсон только мапу может закодировать ?
			JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
	}
}

func JSON(w http.ResponseWriter, status int, v any) error {
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type APIServer struct {
	listenAddr string
	game       *Game
}

func NewAPIServer(addr string, Game *Game) *APIServer {
	return &APIServer{
		listenAddr: addr,
		game:       Game,
	}
}

func (s *APIServer) Run() {
	r := mux.NewRouter()
	r.HandleFunc("/ready", makeHTTPHandleFunc(s.handlePlayerReady))

	//s.game.SetReady()

	http.ListenAndServe(s.listenAddr, r)
}

func (s *APIServer) handlePlayerReady(w http.ResponseWriter, r *http.Request) error {

	s.game.SetReady()
	return JSON(w, http.StatusOK, "ok")
}
