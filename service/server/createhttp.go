//+build:apihttp
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (a *APIKeyCreator) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "json payload required", http.StatusBadRequest)
		return
	}

	var body map[string]string

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, fmt.Sprintf("decoding json payload: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	aud := body["aud"]
	scopes := body["scopes"]
	name := body["display_name"]

	// XXX: TODO allow/deny scopes based on user id in jwt on bearer token used to
	// access this endpoint

	rec, apikey, err := a.create(ctx, name, aud, scopes)
	if err != nil {
		http.Error(w, fmt.Sprintf("error creating key: %v", err.Error()), http.StatusBadRequest)
		return
	}

	resp := struct {
		APIKey string `json:"apikey"`
		ClientRecord
	}{
		ClientRecord: rec,
		APIKey:       apikey,
	}

	json.NewEncoder(w).Encode(resp)

	w.WriteHeader(http.StatusOK)
}
