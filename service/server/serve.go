package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/robinbryce/apikeystore/apibin"
	"google.golang.org/grpc"
)

const (
	ConfigName       = "server"
	DefaultPortHTTP2 = 8402
	DefaultPortHTTP1 = 8401
)

type Config struct {
	Mode          string
	ProjectID     string
	Address1      string
	Address2      string
	Prefix        string
	ExchangeURL   string
	ClientID      string
	ClientSecret  string
	ShutdownGrace time.Duration
	WriteTimeout  time.Duration
	ReadTimeout   time.Duration
	IdleTimeout   time.Duration
}

func NewConfig() Config {
	cfg := Config{
		Mode:          "",
		ProjectID:     "",
		Address1:      fmt.Sprintf("0.0.0.0:%d", DefaultPortHTTP1),
		Address2:      fmt.Sprintf("0.0.0.0:%d", DefaultPortHTTP2),
		Prefix:        "",
		ExchangeURL:   "",
		ClientID:      "",
		ClientSecret:  "",
		ShutdownGrace: time.Second * 15,
		WriteTimeout:  time.Second * 15,
		ReadTimeout:   time.Second * 15,
		IdleTimeout:   time.Second * 60,
	}
	return cfg
}

type Server struct {
	ConfigFileDir string
	cfg           *Config
	clientwriter  APIKeyCreator
	clients       ClientCollection
	authz         APIKeyAuthz
}

type Option func(*Server)

func NewServer(
	ctx context.Context, configFileDir string, cfg *Config, opts ...Option) (Server, error) {

	s := Server{
		ConfigFileDir: configFileDir,
		cfg:           cfg,
	}

	for _, opt := range opts {
		opt(&s)
	}

	s.clientwriter = NewAPIKeyCreator(s.cfg)
	s.clients = NewClientCollection(s.cfg)
	s.authz = NewAPIKeyAuthz(s.cfg)
	return s, nil
}

func normalisePrefixPath(prefix string) (string, error) {
	if prefix == "" {
		return "/", nil
	}

	// Normalise to exactly one leading '/'
	prefix = "/" + strings.TrimLeft(prefix, "/")

	u, err := url.ParseRequestURI(prefix)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(u.Path, "/"), nil
}

func (s *Server) serveGRPC() (func(), func()) {
	listener, err := net.Listen("tcp", s.cfg.Address2)
	if err != nil {
		log.Fatalf("can't start listener: %v", err)
	}

	server := grpc.NewServer(grpc.CustomCodec(flatbuffers.FlatbuffersCodec{}))
	apibin.RegisterAPIKeyStoreServer(server, &s.clientwriter)
	serve := func() {
		if err := server.Serve(listener); err != nil {
			log.Fatalf("failed to start grpc server: %v", err)
		}
	}

	return serve, server.GracefulStop
}

func (s *Server) serveHTTP(root *mux.Router) (func(), func()) {
	// Add your routes as needed
	if root == nil {
		root = mux.NewRouter()
	}

	path, err := normalisePrefixPath(s.cfg.Prefix)
	if err != nil {
		log.Fatalf("bad route prefix: %v", err)
	}

	r := root.PathPrefix(path).Subrouter()
	r.Handle("/clients", &s.clientwriter).Methods("POST", "PUT", "PATCH")
	r.Handle("/clients/{client_id}", &s.clients).Methods("GET")
	r.Handle("/authz/{apikey}", &s.authz).Methods("GET")

	logged := handlers.LoggingHandler(os.Stdout, root)
	srv := &http.Server{
		Addr: s.cfg.Address1,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: s.cfg.WriteTimeout,
		ReadTimeout:  s.cfg.ReadTimeout,
		IdleTimeout:  s.cfg.IdleTimeout,
		Handler:      logged, // Pass our instance of gorilla/mux in.
		// Handler: r, // Pass our instance of gorilla/mux in.
	}

	log.Println("serving:", srv.Addr, "path:", path)

	// Run our server in a goroutine so that it doesn't block.
	serve := func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}

	stop := func() {
		// Create a deadline to wait for.
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownGrace)
		defer cancel()
		// Doesn't block if no connections, but will otherwise wait
		// until the timeout deadline.
		srv.Shutdown(ctx)
		// Optionally, you could run srv.Shutdown in a goroutine and block on
		// <-ctx.Done() if your application should wait for other services
		// to finalize based on context cancellation.
	}
	return serve, stop
}

func (s *Server) Serve() {

	var allstop []func()

	serve, shutdown := s.serveGRPC()
	allstop = append(allstop, shutdown)
	go serve()

	serve, shutdown = s.serveHTTP(nil)
	allstop = append(allstop, shutdown)
	go serve()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	log.Println("shutting down")
	var stopping sync.WaitGroup
	for _, shutdown := range allstop {
		stopping.Add(1)
		go func(shutdown func()) {
			defer stopping.Done()
			shutdown()
		}(shutdown)
	}
	log.Println("clean exit")
	os.Exit(0)
}
