package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/robinbryce/apikeystore/service/keys"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type APIKeyAuthz struct {
	APIKeyHandler
}

func NewAPIKeyAuthz(cfg *Config) APIKeyAuthz {
	return APIKeyAuthz{
		APIKeyHandler: NewAPIKeyHandler(cfg),
	}
}

func (a *APIKeyAuthz) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")

	apikey, ok := mux.Vars(r)[apiKeyRouteVar]
	if !ok {
		http.Error(w, "apikey not found in url", http.StatusForbidden)
		return
	}

	ak, password, err := keys.Decode(apikey)
	if err != nil {
		a.log.Printf("error decoding apikey: %v", err)
		http.Error(w, "invalid api key", http.StatusForbidden)
		return
	}
	key := ak.RecoverKey(password)
	encodedKey := base64.URLEncoding.EncodeToString(key)
	if err != nil {
		a.log.Printf("error decoding key: %v", err)
		http.Error(w, "invalid api key", http.StatusForbidden)
		return
	}

	if err := a.EnsureConnected(ctx); err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to storage: %v", err), http.StatusBadGateway)
		return
	}

	ref := a.db.Collection(apiKeysCollection).Doc(encodedKey)
	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			http.Error(w, fmt.Sprintf("failed to read storage: %v", err), http.StatusBadGateway)
			return
		}
		http.Error(w, "invalid api key", http.StatusForbidden)
		return
	}

	data := doc.Data()
	i, ok := data["aud"]
	aud, ok2 := i.(string)
	if !(ok && ok2) {
		a.log.Printf("error reading key `%s': aud missing: %v", encodedKey, data)
		http.Error(w, "aud missing from record", http.StatusForbidden)
		return
	}

	i, ok = data["scopes"]
	scopes, ok2 := i.(string)
	if !(ok && ok2) {
		a.log.Printf("error reading key `%s': scopes missing: %v", encodedKey, data)
		http.Error(w, "scopes missing from record", http.StatusForbidden)
		return
	}

	// Checking the key here is just a data consistency check. as the derived
	// key is also the database key, the records presence is authorative.
	i, ok = data["key"]
	storedKey, ok2 := i.([]byte)
	if !(ok && ok2) {
		a.log.Printf("error reading key `%s': key missing: %v", encodedKey, data)
		http.Error(w, "key missing from record", http.StatusForbidden)
		return
	}
	if !bytes.Equal(key, storedKey) {
		// treat any discrepancy with the stored record as forbidden
		a.log.Printf("error stored key `%s' != reovered key`%s': %v", storedKey, key, data)
		http.Error(w, "corrupt key record", http.StatusForbidden)

	}

	json.NewEncoder(w).Encode(map[string]string{
		"aud":    aud,
		"scopes": scopes,
	})

	w.WriteHeader(http.StatusOK)
}
