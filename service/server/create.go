package server

import (
	"context"
	"fmt"
	"time"

	"github.com/robinbryce/apikeystore/apibin"
	"github.com/robinbryce/apikeystore/service/keys"
)

const (
	defaultDBTimeout  = time.Second * 30
	apiKeysCollection = "apikeys"
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

func (a *APIKeyCreator) create(
	ctx context.Context, aud, scopes string, opts ...keys.APIKeyOption) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, defaultDBTimeout)
	defer cancel()

	if err := a.EnsureConnected(ctx); err != nil {
		return "", fmt.Errorf("failed to connect to storage: %w", err)
	}

	data := struct {
		keys.APIKey
		Audience string `firestore:"aud"`
		Scopes   string `firestore:"scopes"`
	}{
		Audience: aud,
		Scopes:   scopes,
	}
	err := data.SetOptions(keys.StandardAlg, opts...)
	if err != nil {
		a.log.Printf("error initialising key parameter: %v", err)
		return "", err
	}

	a.log.Printf("display_name: %s\n", data.DisplayName)
	a.log.Printf("audience: %s\n", data.Audience)
	a.log.Printf("scopes: %s\n", data.Scopes)

	apikey, err := data.Generate()
	if err != nil {
		a.log.Printf("error generating key: %v", err)
		return "", err
	}

	// Save the derived key and details in firebase first. The derived key *is*
	// the database primary key (the passwords are properly salted)
	ref := a.db.Collection(apiKeysCollection).Doc(data.EncodedKey())

	// As we just generated this key its essentially impossible for it already
	// to exist unless our key generation is broken - in which case we want to
	// error out.
	_, err = ref.Create(ctx, data)
	if err != nil {
		a.log.Printf("error storing derived key: %v", err)
		return "", err
	}
	return apikey, nil
}
