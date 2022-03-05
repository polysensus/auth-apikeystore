package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	db "cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/robinbryce/apikeystore/apibin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultDBTimeout = time.Second * 30
)

type APIKeyHandler struct {
	cfg      *Config
	log      logger
	firebase *firebase.App
	db       *db.Client
}

type APIKeyCreator struct {
	APIKeyHandler
	apibin.APIKeyStoreServer
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

func (h *APIKeyCreator) ForceReconnect(ctx context.Context) error {
	h.dispose()
	return h.EnsureConnected(ctx)
}

func NewAPIKeyCreator(cfg *Config) APIKeyCreator {
	return APIKeyCreator{
		APIKeyHandler: NewAPIKeyHandler(cfg),
	}
}

func (a *APIKeyCreator) Create(
	ctx context.Context, in *apibin.CreateRequest) (*flatbuffers.Builder, error) {

	ctx, cancel := context.WithTimeout(ctx, defaultDBTimeout)
	defer cancel()

	if err := a.EnsureConnected(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to storage: %w", err)
	}

	a.log.Printf("display_name: %s\n", string(in.DisplayName()))
	a.log.Printf("audience: %s\n", string(in.Audience()))
	a.log.Printf("scopes: %s\n", string(in.Scopes()))

	b := flatbuffers.NewBuilder(0)
	apikey := b.CreateString("todo")
	apibin.CreateResultStart(b)
	apibin.CreateResultAddApikey(b, apikey)
	b.Finish(apibin.CreateResultEnd(b))
	return b, nil
}

func (a *APIKeyCreator) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()

	if err := a.EnsureConnected(ctx); err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to storage: %v", err), http.StatusBadGateway)
		return
	}

	docname := "Hello1"
	var doc map[string]interface{}

	a.log.Printf("%v, %v\n", a.firebase, a.db)

	hello := a.db.Doc("hellos/hello1")
	a.log.Printf("%v\n", hello)
	d, err := hello.Get(ctx)
	if err != nil {

		if status.Code(err) != codes.NotFound {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		doc = map[string]interface{}{"name": docname}

		_, err := hello.Set(ctx, doc)
		if err != nil {
			a.log.Printf("failed to Set: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		a.log.Printf("created: %v\n", doc)
		w.WriteHeader(http.StatusOK)
		return
	}

	doc = d.Data()

	a.log.Printf("got %v\n", doc)
	w.WriteHeader(http.StatusOK)
}
