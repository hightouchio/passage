package tunnel

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/postgres"
	"net/http"
)

func (s Server) ConfigureWebRoutes(router *mux.Router) {
	router.HandleFunc("/tunnel/reverse", s.handleWebNewReverseTunnel).Methods(http.MethodPost)
	//router.HandleFunc("/tunnel/normal", nil).Methods(http.MethodPost)

	tunnelRouter := router.PathPrefix("/tunnel/{tunnelID}").Subrouter()
	tunnelRouter.HandleFunc("", s.handleWebTunnelGet).Methods(http.MethodGet)
	tunnelRouter.HandleFunc("/check", s.handleWebTunnelCheck).Methods(http.MethodGet)
}

func (s Server) handleWebTunnelGet(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger(r.Context())

	var request GetConnectionDetailsRequest
	if err := getTunnelID(r, &request.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.GetConnectionDetails(r.Context(), request)
	defer log.Request(logger, "tunnel:GetConnectionDetails", request, response, err)
	if err != nil {
		switch err {
		case postgres.ErrTunnelNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	renderJSON(w, response)
}

func (s Server) handleWebTunnelCheck(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger(r.Context())

	var request CheckTunnelRequest
	if err := getTunnelID(r, &request.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.CheckTunnel(r.Context(), request)
	defer log.Request(logger, "tunnel:Check", request, response, err)

	if err != nil {
		switch err {
		case postgres.ErrTunnelNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	renderJSON(w, response)
}

func (s Server) handleWebNewReverseTunnel(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger(r.Context())

	var request NewReverseTunnelRequest
	if err := read(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.NewReverseTunnel(r.Context(), request)
	defer log.Request(logger, "tunnel:NewReverse", response, request, err)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, response)
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

func renderJSON(w http.ResponseWriter, ret interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}
