package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	clientIDRouteVar = "client_id"
)

type ClientCollection struct {
	APIKeyHandler
}

func NewClientCollection(cfg *Config) ClientCollection {
	return ClientCollection{
		APIKeyHandler: NewAPIKeyHandler(cfg),
	}
}

func (cc *ClientCollection) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")

	clientID, ok := mux.Vars(r)[clientIDRouteVar]
	if !ok {
		http.Error(w, "client_id not found in url", http.StatusBadRequest)
		return
	}

	if err := cc.EnsureConnected(ctx); err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to storage: %v", err), http.StatusBadGateway)
		return
	}

	ref := cc.db.Collection(cc.cfg.ClientCollectionID).Doc(clientID)

	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			http.Error(w, fmt.Sprintf("failed to read storage: %v", err), http.StatusBadGateway)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var rec ClientRecord
	doc.DataTo(&rec)
	json.NewEncoder(w).Encode(rec)

	w.WriteHeader(http.StatusOK)
}
