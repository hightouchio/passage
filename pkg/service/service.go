package service

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/pkg/tunnels"
)

type Service struct {
	tunnels        *tunnels.Tunnels
	reverseTunnels *tunnels.ReverseTunnels
	router         *mux.Router
}

func NewService(
	tunnels *tunnels.Tunnels,
	reverseTunnels *tunnels.ReverseTunnels,
) *Service {
	s := &Service{
		tunnels:        tunnels,
		reverseTunnels: reverseTunnels,
		router:         mux.NewRouter(),
	}

	apiRouter := s.router.PathPrefix("/api").Subrouter()

	apiRouter.HandleFunc("/tunnels", s.createTunnel).Methods("POST")
	apiRouter.HandleFunc("/tunnels/{id}", s.getTunnel).Methods("GET")
	apiRouter.HandleFunc("/tunnels", s.listTunnels).Methods("GET")

	return s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Service) createTunnel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID              string `json:"id"`
		ServiceEndpoint string `json:"serviceEndpoint"`
		ServicePort     uint32 `json:"servicePort"`
	}
	if err := read(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tunnel, err := s.tunnels.Create(r.Context(), req.ID, req.ServiceEndpoint, req.ServicePort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, tunnel)
}

func (s *Service) getTunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	tunnel, err := s.tunnels.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, tunnel)
}

func (s *Service) listTunnels(w http.ResponseWriter, r *http.Request) {
	tunnels, err := s.tunnels.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, tunnels)
}

func read(r *http.Request, req interface{}) error {
	return json.NewDecoder(r.Body).Decode(&req)
}

func respond(w http.ResponseWriter, ret interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}
