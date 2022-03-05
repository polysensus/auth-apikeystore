package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/robinbryce/apikeystore/service/keys"
)

func (a *APIKeyCreator) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "json payload required", http.StatusBadRequest)
		return
	}

	var data map[string]string

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, fmt.Sprintf("decoding json payload: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	aud := data["aud"]
	scopes := data["scopes"]
	name := data["display_name"]

	var opts []keys.APIKeyOption
	if name != "" {
		opts = append(opts, keys.WithDisplayName(name))
	}

	apikey, err := a.create(ctx, aud, scopes, opts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("error creating key: %v", err.Error()), http.StatusBadRequest)
		return
	}

	// XXX: TODO allow/deny scopes based on user id in jwt on bearer token used to
	// access this endpoint
	json.NewEncoder(w).Encode(map[string]string{
		"apikey": apikey,
		"aud":    aud,
		"scopes": scopes,
	})

	w.WriteHeader(http.StatusOK)
}
