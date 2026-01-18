// Package server provides functionality to easily set up an HTTTP server.
//
// The server holds all the clients it needs and they should be set up in the Create method.
//
// The HTTP routes and middleware are set up in the setupRouter method.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/client/database"
	"github.com/zoff-music/vibes/client/internalpubsub"
	"github.com/zoff-music/vibes/client/youtube"
	"github.com/zoff-music/vibes/config"
	"github.com/zoff-music/vibes/monitoring/metrics"
	"github.com/zoff-music/vibes/monitoring/trace"
	"github.com/zoff-music/vibes/server/internal/event"
)

// Server holds the HTTP server, router, config and all clients.
type Server struct {
	Config         *config.Config
	HTTP           *http.Server
	DB             *database.Client
	InternalPubSub *internalpubsub.Client
	YouTube        *youtube.Client
	Router         *mux.Router
}

// Create sets up the HTTP server, router and all clients.
// Returns an error if an error occurs.
func (s *Server) Create(ctx context.Context, config *config.Config) error {
	metrics.RegisterPrometheusCollectors()

	var internalpubsubClient internalpubsub.Client
	err := internalpubsubClient.Init()
	if err != nil {
		return fmt.Errorf("internalpubsub client: %w", err)
	}

	var dbClient database.Client
	err = dbClient.Init(ctx, config)
	if err != nil {
		return fmt.Errorf("database client: %w", err)
	}

	var youtubeClient youtube.Client
	err = youtubeClient.Init(config.YouTubeAPIKey)
	if err != nil {
		return fmt.Errorf("youtube client: %w", err)
	}

	s.Config = config
	s.DB = &dbClient
	s.InternalPubSub = &internalpubsubClient
	s.YouTube = &youtubeClient
	s.Router = mux.NewRouter()
	s.HTTP = &http.Server{
		Addr:              fmt.Sprintf(":%s", s.Config.Port),
		Handler:           s.Router,
		ReadHeaderTimeout: 2 * time.Second, // prevent slowloris attacks
	}

	s.setupRoutes()

	return nil
}

// Serve starts subscribing for messages.
// It also makes sure that the server gracefully shuts down on exit.
// Returns an error if an error occurs.
func (s *Server) Serve(ctx context.Context, errc chan<- error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	closer, err := trace.InitGlobalTracer(s.Config)
	if err != nil {
		errc <- err
	}

	defer closer.Close()

	go s.serveHTTP(ctx, errc)
	go s.subscribeAndListen(ctx, errc)

	log.Info("Ready")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until we receive sigterm or interrupt
	<-stop

	log.Info("Main server has received a shutdown signal")
	cancel() // Stop background jobs

	s.shutdown(ctx)
}

func (s *Server) serveHTTP(ctx context.Context, errc chan<- error) {
	go func(ctx context.Context, httpServ *http.Server) {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		// Block until we receive sigterm or interrupt
		<-stop

		log.Info("HTTP server has received a shutdown signal")

		if err := httpServ.Shutdown(ctx); err != nil {
			log.Error(err.Error())
		}
	}(ctx, s.HTTP)

	log.Infof("Ready at: %s", s.Config.Port)

	// Block until httpServ.Shutdown is called
	if err := s.HTTP.ListenAndServe(); err != http.ErrServerClosed {
		errc <- fmt.Errorf("unexpected server error: %w", err)
		return
	}

	log.Info("HTTP server closed")
}

func (s *Server) subscribeAndListen(ctx context.Context, errc chan<- error) {
	for _, e := range event.GetAppEvents(s.DB, s.DB, s.InternalPubSub, s.DB) {
		go func(e event.AppEvent) {
			e.SubscribeAndListen(ctx)
		}(e)
	}
}

func (s *Server) shutdown(ctx context.Context) {
	if s.DB != nil {
		err := s.DB.Close()
		if err != nil {
			log.Errorf("error closing database: %v", err)
		}
	}

	log.Info("client closed")
}
