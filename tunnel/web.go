package tunnel

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/tunnel/postgres"
	"net/http"
)

func (s Server) ConfigureWebRoutes(router *mux.Router) {
	router.HandleFunc("/tunnel/reverse", s.handleWebNewReverseTunnel).Methods(http.MethodPost)
	//router.HandleFunc("/tunnel/normal", nil).Methods(http.MethodPost)

	tunnelRouter := router.PathPrefix("/tunnel/{tunnelID}").Subrouter()
	tunnelRouter.HandleFunc("", s.handleWebGetConnectionDetails).Methods(http.MethodGet)
	tunnelRouter.HandleFunc("/check", s.handleWebCheckTunnel).Methods(http.MethodGet)
}

func (s Server) handleWebGetConnectionDetails(w http.ResponseWriter, r *http.Request) {
	var id uuid.UUID
	if err := getTunnelID(r, &id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := s.GetConnectionDetails(r.Context(), GetConnectionDetailsRequest{id})
	if err != nil {
		switch err {
		case postgres.ErrTunnelNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	respond(w, res)
}

func (s Server) handleWebCheckTunnel(w http.ResponseWriter, r *http.Request) {
	var id uuid.UUID
	if err := getTunnelID(r, &id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := s.CheckTunnel(r.Context(), CheckTunnelRequest{id})
	if err != nil {
		switch err {
		case postgres.ErrTunnelNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	respond(w, res)
}

func (s Server) handleWebNewReverseTunnel(w http.ResponseWriter, r *http.Request) {
	var req NewReverseTunnelRequest
	if err := read(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.NewReverseTunnel(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, response)
}

func getTunnelID(r *http.Request, id *uuid.UUID) error {
	tunnelID, ok := mux.Vars(r)["tunnelID"]
	if !ok {
		return errors.New("no id specified")
	}

	var err error
	*id, err = uuid.Parse(tunnelID)
	if err != nil {
		return errors.New("could not parse id (must be valid UUID v4)")
	}

	return nil
}

func read(r *http.Request, req interface{}) error {
	return json.NewDecoder(r.Body).Decode(&req)
}

func respond(w http.ResponseWriter, ret interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}
