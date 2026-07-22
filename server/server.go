// Package server provides functionality to easily set up an HTTTP server.
//
// The server holds all the clients it needs and they should be set up in the Create method.
//
// The HTTP routes and middleware are set up in the setupRouter method.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client/database"
	"github.com/zoff-music/vibes-backend/client/internalpubsub"
	"github.com/zoff-music/vibes-backend/client/soundcloud"
	"github.com/zoff-music/vibes-backend/client/spotify"
	"github.com/zoff-music/vibes-backend/client/youtube"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/metrics"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/server/internal/event"
)

// Server holds the HTTP server, router, config and all clients.
type Server struct {
	Config         *config.Config
	HTTP           *http.Server
	InternalHTTP   *http.Server
	DB             *database.Client
	InternalPubSub *internalpubsub.Client
	YouTube        *youtube.Client
	SoundCloud     *soundcloud.Client
	Spotify        *spotify.Client
	Router         *mux.Router
	InternalRouter *mux.Router
}

// Create sets up the HTTP server, router and all clients.
// Returns an error if an error occurs.
func (s *Server) Create(ctx context.Context, config *config.Config) error {
	metrics.RegisterPrometheusCollectors()

	var internalpubsubClient internalpubsub.Client
	err := internalpubsubClient.Init()
	if err != nil {
		return fmt.Errorf("error initializing internalpubsub client: %w", err)
	}

	var dbClient database.Client
	err = dbClient.Init(ctx, config)
	if err != nil {
		return fmt.Errorf("error initializing database client: %w", err)
	}

	var youtubeClient youtube.Client
	err = youtubeClient.Init(ctx, config)
	if err != nil {
		return fmt.Errorf("error initializing youtube client: %w", err)
	}

	var soundcloudClient soundcloud.Client
	err = soundcloudClient.Init(ctx, config)
	if err != nil {
		return fmt.Errorf("error initializing soundcloud client: %w", err)
	}

	var spotifyClient spotify.Client
	err = spotifyClient.Init(ctx, config)
	if err != nil {
		return fmt.Errorf("error initializing spotify client: %w", err)
	}

	s.Config = config
	s.DB = &dbClient
	s.InternalPubSub = &internalpubsubClient
	s.YouTube = &youtubeClient
	s.SoundCloud = &soundcloudClient
	s.Spotify = &spotifyClient
	s.Router = mux.NewRouter()
	s.InternalRouter = mux.NewRouter()
	s.HTTP = &http.Server{
		Addr:              fmt.Sprintf(":%s", s.Config.Port),
		Handler:           s.Router,
		ReadHeaderTimeout: 2 * time.Second, // prevent slowloris attacks
	}
	s.InternalHTTP = &http.Server{
		Addr:              fmt.Sprintf(":%s", s.Config.InternalPort),
		Handler:           s.InternalRouter,
		ReadHeaderTimeout: 2 * time.Second,
	}

	s.setupRoutes()
	s.setupInternalRoutes()

	return nil
}

// Serve starts subscribing for messages.
// It also makes sure that the server gracefully shuts down on exit.
// Returns an error if an error occurs.
func (s *Server) Serve(ctx context.Context, errc chan<- error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	closer, err := tracing.Init(s.Config)
	if err != nil {
		errc <- err
		return
	}

	defer closer.Close()

	go s.serveHTTP(ctx, errc)
	go s.serveInternalHTTP(ctx, errc)
	go s.subscribeAndListen(ctx, errc)

	log.Println("Ready")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until we receive sigterm or interrupt
	<-stop

	log.Println("Main server has received a shutdown signal")
	cancel() // Stop background jobs

	s.shutdown(ctx)
}

func (s *Server) serveInternalHTTP(ctx context.Context, errc chan<- error) {
	go func(ctx context.Context, httpServ *http.Server) {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		// Block until we receive sigterm or interrupt
		<-stop

		log.Println("Internal HTTP server has received a shutdown signal")

		err := httpServ.Shutdown(ctx)
		if err != nil {
			log.Println(err.Error())
		}
	}(ctx, s.InternalHTTP)

	log.Printf("Internal ready at: %s", s.Config.InternalPort)

	// Block until httpServ.Shutdown is called
	err := s.InternalHTTP.ListenAndServe()
	if err != http.ErrServerClosed {
		errc <- fmt.Errorf("error unexpected internal server error: %w", err)
		return
	}

	log.Println("Internal HTTP server closed")
}

func (s *Server) serveHTTP(ctx context.Context, errc chan<- error) {
	go func(ctx context.Context, httpServ *http.Server) {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		// Block until we receive sigterm or interrupt
		<-stop

		log.Println("HTTP server has received a shutdown signal")

		err := httpServ.Shutdown(ctx)
		if err != nil {
			log.Println(err.Error())
		}
	}(ctx, s.HTTP)

	log.Printf("Ready at: %s", s.Config.Port)

	// Block until httpServ.Shutdown is called
	err := s.HTTP.ListenAndServe()
	if err != http.ErrServerClosed {
		errc <- fmt.Errorf("error unexpected server error: %w", err)
		return
	}

	log.Println("HTTP server closed")
}

func (s *Server) subscribeAndListen(ctx context.Context, errc chan<- error) {
	for _, e := range event.GetAppEvents(s.DB, s.InternalPubSub, s.Spotify, s.YouTube) {
		go func(e event.AppEvent) {
			e.SubscribeAndListen(ctx)
		}(e)
	}
}

func (s *Server) shutdown(ctx context.Context) {
	if s.DB != nil {
		err := s.DB.Close()
		if err != nil {
			log.Printf("error closing database: %v", err)
		}
	}

	log.Println("client closed")
}
