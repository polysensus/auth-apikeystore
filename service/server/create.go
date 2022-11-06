package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/robinbryce/apikeys"
	"github.com/polysensus/auth-apikeystore/apibin"
)

const (
	defaultDBTimeout  = time.Second * 30
	apiKeysCollection = "apiclients"
	apiKeyRouteVar    = "apikey"
)

type APIKeyCreator struct {
	APIKeyHandler
	apibin.APIKeyStoreServer
}

func NewAPIKeyCreator(cfg *Config) APIKeyCreator {
	return APIKeyCreator{
		APIKeyHandler: NewAPIKeyHandler(cfg),
	}
}

type ClientRecord struct {
	apikeys.Key
	DisplayName string `firestore:"display_name"`
	Audience    string `firestore:"aud"`
	Scope       string `firestore:"scope"`
}

// MarshalJSON ensures that derived_key is marshaled as url safe form for consistency with how the parts of the apikeys are serialized
func (cr *ClientRecord) MarshalJSON() ([]byte, error) {
	type Alias ClientRecord
	return json.Marshal(&struct {
		DerivedKey string `json:"derived_key"`
		*Alias
	}{
		DerivedKey: cr.EncodedKey(),
		Alias:      (*Alias)(cr),
	})
}

func (cr *ClientRecord) UnmarshalJSON(data []byte) error {
	type Alias ClientRecord
	aux := &struct {
		DerivedKey string `json:"derived_key"`
		*Alias
	}{
		Alias: (*Alias)(cr),
	}

	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	cr.DerivedKey, err = base64.URLEncoding.DecodeString(aux.DerivedKey)
	if err != nil {
		return err
	}
	return nil
}

func (a *APIKeyCreator) create(
	ctx context.Context, displayName, aud, scopes string, opts ...apikeys.KeyOption) (ClientRecord, string, error) {

	ctx, cancel := context.WithTimeout(ctx, defaultDBTimeout)
	defer cancel()

	if err := a.EnsureConnected(ctx); err != nil {
		return ClientRecord{}, "", fmt.Errorf("failed to connect to storage: %w", err)
	}

	rec := ClientRecord{
		DisplayName: displayName,
		Audience:    aud,
		Scope:       scopes,
	}
	err := rec.SetOptions(apikeys.StandardAlg, opts...)
	if err != nil {
		a.log.Printf("error initialising key parameter: %v", err)
		return ClientRecord{}, "", err
	}

	apikey, err := rec.Generate()
	if err != nil {
		a.log.Printf("error generating key: %v", err)
		return ClientRecord{}, "", err
	}

	// Use the client_id as the primary key
	ref := a.db.Collection(apiKeysCollection).Doc(rec.ClientID)

	// As we just generated this key its essentially impossible for it already
	// to exist unless our key generation is broken - in which case we want to
	// error out.
	_, err = ref.Create(ctx, rec)
	if err != nil {
		a.log.Printf("error storing derived key: %v", err)
		return ClientRecord{}, "", err
	}
	return rec, apikey, nil
}
