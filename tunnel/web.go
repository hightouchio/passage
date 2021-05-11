package tunnel

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

func (s Server) ConfigureWebRoutes(router *mux.Router) {
	router.HandleFunc("/reverse_tunnels", s.handleWebNewReverseTunnel).Methods(http.MethodPost)
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

func (s Server) handleWebGetReverseTunnel(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	//id := vars["id"]
	//
	//tunnel, err := s.tunnels.Get(r.Context(), id)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//
	//respond(w, tunnel)
}

func (s Server) handleWebListReverseTunnels(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	//id := vars["id"]
	//
	//tunnel, err := s.tunnels.Get(r.Context(), id)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//
	//respond(w, tunnel)
}

func read(r *http.Request, req interface{}) error {
	return json.NewDecoder(r.Body).Decode(&req)
}

func respond(w http.ResponseWriter, ret interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}
