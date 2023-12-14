package tunnel

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/tunnel/postgres"
	"net/http"
)

func (s API) ConfigureWebRoutes(router *mux.Router) {
	// Create tunnel endpoints.
	router.HandleFunc("/tunnel/normal", s.handleWebCreateNormalTunnel).Methods(http.MethodPost)
	router.HandleFunc("/tunnel/reverse", s.handleWebCreateReverseTunnel).Methods(http.MethodPost)

	tunnelRouter := router.PathPrefix("/tunnel/{tunnelID}").Subrouter()
	tunnelRouter.HandleFunc("", s.handleWebTunnelGet).Methods(http.MethodGet)
	tunnelRouter.HandleFunc("/check", s.handleWebTunnelCheck).Methods(http.MethodGet)
	tunnelRouter.HandleFunc("", s.handleWebTunnelUpdate).Methods(http.MethodPut)
	tunnelRouter.HandleFunc("", s.handleWebTunnelDelete).Methods(http.MethodDelete)
}

func (s API) handleWebTunnelGet(w http.ResponseWriter, r *http.Request) {
	var request GetTunnelRequest
	if err := getTunnelID(r, &request.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.GetTunnel(r.Context(), request)
	if err != nil {
		switch err {
		case postgres.ErrTunnelNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			setRequestError(r, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	renderJSON(w, response)
}

func (s API) handleWebTunnelCheck(w http.ResponseWriter, r *http.Request) {
	var request CheckTunnelRequest
	if err := getTunnelID(r, &request.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.CheckTunnel(r.Context(), request)
	if err != nil {
		switch err {
		case postgres.ErrTunnelNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			setRequestError(r, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	renderJSON(w, response)
}

func (s API) handleWebTunnelUpdate(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		setRequestError(r, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, response)
}

func (s API) handleWebTunnelDelete(w http.ResponseWriter, r *http.Request) {
	var request DeleteTunnelRequest
	if err := getTunnelID(r, &request.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.DeleteTunnel(r.Context(), request)
	if err != nil {
		switch err {
		case postgres.ErrTunnelNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			setRequestError(r, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	renderJSON(w, response)
}

func (s API) handleWebCreateNormalTunnel(w http.ResponseWriter, r *http.Request) {
	var request CreateNormalTunnelRequest
	if err := read(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.CreateNormalTunnel(r.Context(), request)
	if err != nil {
		setRequestError(r, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, response)
}

func (s API) handleWebCreateReverseTunnel(w http.ResponseWriter, r *http.Request) {
	var request CreateReverseTunnelRequest
	if err := read(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.CreateReverseTunnel(r.Context(), request)
	if err != nil {
		setRequestError(r, err)
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

func setRequestError(r *http.Request, err error) {
	errorFunc, ok := r.Context().Value("_set_error_func").(func(error))
	if !ok {
		return
	}
	errorFunc(err)
}

func renderJSON(w http.ResponseWriter, ret interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ret)
}
