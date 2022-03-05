package server

import (
	"context"
	"log"

	db "cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
)

type APIKeyHandler struct {
	cfg      *Config
	log      logger
	firebase *firebase.App
	db       *db.Client
}

func NewAPIKeyHandler(cfg *Config) APIKeyHandler {

	return APIKeyHandler{
		cfg: cfg,
		log: log.Default(),
	}
}

func (h *APIKeyHandler) dispose() {

	if h.db != nil {
		h.db = nil
	}
	if h.firebase != nil {
		h.firebase = nil
	}
}

func (h *APIKeyHandler) isConnected() bool {
	if h.db == nil || h.firebase == nil {
		return false
	}
	return true
}

func (h *APIKeyHandler) EnsureConnected(ctx context.Context) error {
	var err error

	if h.isConnected() {
		return nil
	}

	h.log.Printf("connecting\n")

	h.firebase, err = firebase.NewApp(ctx, &firebase.Config{
		ProjectID: h.cfg.ProjectID})
	if err != nil {
		h.dispose()
		return err
	}

	h.db, err = h.firebase.Firestore(ctx)
	if err != nil {
		h.dispose()
		return err
	}
	h.log.Printf("connected: %v %v\n", h.firebase, h.db)
	return nil
}

func (h *APIKeyHandler) ForceReconnect(ctx context.Context) error {
	h.dispose()
	return h.EnsureConnected(ctx)
}
