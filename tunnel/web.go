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

func (s API) ConfigureWebRoutes(router *mux.Router) {
	// create tunnel
	router.HandleFunc("/tunnel/standard", s.handleWebCreateStandardTunnel).Methods(http.MethodPost)
	router.HandleFunc("/tunnel/reverse", s.handleWebCreateReverseTunnel).Methods(http.MethodPost)

	tunnelRouter := router.PathPrefix("/tunnel/{tunnelID}").Subrouter()
	tunnelRouter.HandleFunc("", s.handleWebTunnelGet).Methods(http.MethodGet)
	tunnelRouter.HandleFunc("/check", s.handleWebTunnelCheck).Methods(http.MethodGet)
	tunnelRouter.HandleFunc("", s.handleWebTunnelUpdate).Methods(http.MethodPut)
	tunnelRouter.HandleFunc("", s.handleWebTunnelDelete).Methods(http.MethodDelete)
}

func (s API) handleWebTunnelGet(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger(r.Context())

	var request GetTunnelRequest
	if err := getTunnelID(r, &request.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.GetTunnel(r.Context(), request)
	defer log.Request(logger, "tunnel:Get", request, response, err)
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

func (s API) handleWebTunnelCheck(w http.ResponseWriter, r *http.Request) {
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

func (s API) handleWebTunnelUpdate(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger(r.Context())

	var request UpdateTunnelRequest
	if err := getTunnelID(r, &request.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := read(r, &request.UpdateFields); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.UpdateTunnel(r.Context(), request)
	defer log.Request(logger, "tunnel:Update", response, request, err)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, response)
}

func (s API) handleWebTunnelDelete(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger(r.Context())

	var request DeleteTunnelRequest
	if err := getTunnelID(r, &request.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.DeleteTunnel(r.Context(), request)
	defer log.Request(logger, "tunnel:Delete", request, response, err)

	if err != nil {
		switch err {
		case postgres.ErrTunnelNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
}

func (s API) handleWebCreateStandardTunnel(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger(r.Context())

	var request CreateStandardTunnelRequest
	if err := read(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.CreateStandardTunnel(r.Context(), request)
	defer log.Request(logger, "tunnel:CreateStandard", response, request, err)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, response)
}

func (s API) handleWebCreateReverseTunnel(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger(r.Context())

	var request CreateReverseTunnelRequest
	if err := read(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.CreateReverseTunnel(r.Context(), request)
	defer log.Request(logger, "tunnel:CreateReverse", response, request, err)

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
