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
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/robinbryce/apikeystore/apibin"
	"google.golang.org/grpc"
)

const (
	ConfigName  = "server"
	DefaultPort = 8401
)

type Config struct {
	Mode          string
	ProjectID     string
	Address       string
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
		Address:       fmt.Sprintf("0.0.0.0:%d", DefaultPort),
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
	writer        APIKeyCreator
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

	s.writer = NewAPIKeyCreator(s.cfg)
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
	return u.Path, nil
}

func (s *Server) serveGRPC() (func(), func()) {
	listener, err := net.Listen("tcp", s.cfg.Address)
	if err != nil {
		log.Fatalf("can't start listener: %v", err)
	}

	server := grpc.NewServer(grpc.CustomCodec(flatbuffers.FlatbuffersCodec{}))
	apibin.RegisterAPIKeyStoreServer(server, &s.writer)
	serve := func() {
		if err := server.Serve(listener); err != nil {
			log.Fatalf("failed to start grpc server: %v", err)
		}
	}

	return serve, server.GracefulStop
}

func (s *Server) serveHTTP() (func(), func()) {
	// Add your routes as needed
	r := mux.NewRouter()

	path, err := normalisePrefixPath(s.cfg.Prefix)
	if err != nil {
		log.Fatalf("bad route prefix: %v", err)
	}

	path = fmt.Sprintf("%screate", path)
	r.PathPrefix(path).Handler(&s.writer)

	logged := handlers.LoggingHandler(os.Stdout, r)

	srv := &http.Server{
		Addr: s.cfg.Address,
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

	serve, shutdown := s.serveGRPC()
	go serve()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	shutdown()

	log.Println("shutting down")
	os.Exit(0)
}
