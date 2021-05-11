package service

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/pkg/ssh"
	"github.com/hightouchio/passage/pkg/tunnels"
	"github.com/pkg/errors"
	"net/http"
)

type Service struct {
	tunnels        *tunnels.Tunnels
	reverseTunnels *tunnels.ReverseTunnels
	router         *mux.Router
}

func NewService(tunnels *tunnels.Tunnels, reverseTunnels *tunnels.ReverseTunnels) *Service {
	s := &Service{
		tunnels:        tunnels,
		reverseTunnels: reverseTunnels,
		router:         mux.NewRouter(),
	}

	apiRouter := s.router.PathPrefix("/api").Subrouter()

	apiRouter.HandleFunc("/tunnels", s.createTunnel).Methods("POST")
	apiRouter.HandleFunc("/tunnels/{id}", s.getTunnel).Methods("GET")
	apiRouter.HandleFunc("/tunnels", s.listTunnels).Methods("GET")

	apiRouter.HandleFunc("/reverse_tunnels", s.handleWebCreateReverseTunnel).Methods("POST")

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

	// we always generate keys for normal tunnels because we are initiating an outbound connection to the customer
	keys, err := ssh.GenerateKeyPair()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	tunnel, err := s.tunnels.Create(r.Context(), req.ID, req.ServiceEndpoint, req.ServicePort, keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, tunnel)
}

type createReverseTunnelRequest struct {
	PublicKey string `json:"publicKey"`
}

type createReverseTunnelResponse struct {
	ID         int     `json:"id"`
	PrivateKey *string `json:"privateKeyBase64,omitempty"`
}

func (s *Service) handleWebCreateReverseTunnel(w http.ResponseWriter, r *http.Request) {
	var req createReverseTunnelRequest
	if err := read(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.createReverseTunnel(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, response)
}

func (s *Service) createReverseTunnel(ctx context.Context, req createReverseTunnelRequest) (*createReverseTunnelResponse, error) {
	// check if we need to generate a new keypair or can just use what the customer provided
	var keys ssh.KeyPair
	if req.PublicKey != "" {
		if !ssh.IsValidPublicKey(req.PublicKey) {
			return nil, errors.New("invalid public key")
		}

		keys = ssh.KeyPair{PublicKey: req.PublicKey}
	} else {
		var err error
		keys, err = ssh.GenerateKeyPair()
		if err != nil {
			return nil, errors.New("could not generate key pair")
		}
	}

	tunnel, err := s.reverseTunnels.Create(ctx, keys)
	if err != nil {
		return nil, errors.Wrap(err, "could not create reverse tunnel")
	}

	response := &createReverseTunnelResponse{ID: tunnel.ID}
	// attach private key if necessary
	if keys.PrivateKey != "" {
		b64 := keys.Base64PrivateKey()
		response.PrivateKey = &b64
	}
	return response, nil
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
