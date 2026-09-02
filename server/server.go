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
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client/database"
	"github.com/zoff-music/vibes-backend/client/gemini"
	"github.com/zoff-music/vibes-backend/client/grok"
	redisclient "github.com/zoff-music/vibes-backend/client/redis"
	"github.com/zoff-music/vibes-backend/client/soundcloud"
	"github.com/zoff-music/vibes-backend/client/youtube"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/metrics"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/server/internal/event"
	"github.com/zoff-music/vibes-backend/server/internal/middleware"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Server holds the HTTP server, router, config and all clients.
type Server struct {
	Config         *config.Config
	HTTP           *http.Server
	InternalHTTP   *http.Server
	DB             *database.Client
	Redis          *redisclient.Client
	YouTube        *youtube.Client
	SoundCloud     *soundcloud.Client
	AI             vibe.PlaylistGenerator
	Router         *mux.Router
	InternalRouter *mux.Router
}

// Create sets up the HTTP server, router and all clients.
// Returns an error if an error occurs.
func (s *Server) Create(ctx context.Context, config *config.Config) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Create")
	defer span.End()

	metrics.RegisterPrometheusCollectors()

	var redisClient redisclient.Client
	err := redisClient.Init(ctx, config)
	if err != nil {
		return fmt.Errorf("error initializing redis client: %w", err)
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

	var grokClient grok.Client
	err = grokClient.Init(ctx, config)
	if err != nil {
		return fmt.Errorf("error initializing grok client: %w", err)
	}

	var geminiClient gemini.Client
	err = geminiClient.Init(ctx, config)
	if err != nil {
		return fmt.Errorf("error initializing gemini client: %w", err)
	}

	aiModel, err := vibe.ParseAIModel(config.AIModel)
	if err != nil {
		return fmt.Errorf("error parsing configured AI model: %w", err)
	}

	var ai vibe.PlaylistGenerator
	switch aiModel.Provider {
	case vibe.AIProviderGrok:
		ai = &grokClient
	case vibe.AIProviderGemini:
		ai = &geminiClient
	}

	s.Config = config
	s.DB = &dbClient
	s.Redis = &redisClient
	s.YouTube = &youtubeClient
	s.SoundCloud = &soundcloudClient
	s.AI = ai
	s.Router = mux.NewRouter()
	s.InternalRouter = mux.NewRouter()
	s.HTTP = &http.Server{
		Addr:              fmt.Sprintf(":%s", s.Config.Port),
		Handler:           middleware.CompressionMiddleware(s.Router),
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
		errc <- fmt.Errorf("error initializing tracing: %w", err)
		return
	}

	defer closer.Close()

	s.HTTP.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	s.InternalHTTP.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	serverErrors := make(chan error, 2)
	var running sync.WaitGroup
	running.Add(2)
	go func() {
		defer running.Done()
		s.serveHTTP(serverErrors)
	}()
	go func() {
		defer running.Done()
		s.serveInternalHTTP(serverErrors)
	}()
	s.subscribeAndListen(ctx, &running)

	log.Println("Ready")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	var serveErr error
	select {
	case <-stop:
		log.Println("Main server has received a shutdown signal")
	case serveErr = <-serverErrors:
		log.Printf("Main server has received a server error: %v", serveErr)
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	s.shutdownHTTP(shutdownCtx)
	shutdownCancel()
	running.Wait()
	s.shutdown(context.Background())

	errc <- serveErr
}

func (s *Server) serveInternalHTTP(errc chan<- error) {
	log.Printf("Internal ready at: %s", s.Config.InternalPort)

	err := s.InternalHTTP.ListenAndServe()
	if err != http.ErrServerClosed {
		errc <- fmt.Errorf("error unexpected internal server error: %w", err)
		return
	}

	log.Println("Internal HTTP server closed")
}

func (s *Server) serveHTTP(errc chan<- error) {
	log.Printf("Ready at: %s", s.Config.Port)

	err := s.HTTP.ListenAndServe()
	if err != http.ErrServerClosed {
		errc <- fmt.Errorf("error unexpected server error: %w", err)
		return
	}

	log.Println("HTTP server closed")
}

func (s *Server) subscribeAndListen(ctx context.Context, running *sync.WaitGroup) {
	for _, e := range event.GetAppEvents(
		s.DB,
		s.Redis,
		s.SoundCloud,
		s.YouTube,
		s.AI,
		s.Config.EnabledProviders(),
	) {
		running.Add(1)
		go func(e event.AppEvent) {
			defer running.Done()
			e.SubscribeAndListen(ctx)
		}(e)
	}
}

func (s *Server) shutdownHTTP(ctx context.Context) {
	err := s.HTTP.Shutdown(ctx)
	if err != nil {
		log.Printf("error shutting down HTTP server: %v", err)
		closeErr := s.HTTP.Close()
		if closeErr != nil {
			log.Printf("error closing HTTP server: %v", closeErr)
		}
	}

	err = s.InternalHTTP.Shutdown(ctx)
	if err != nil {
		log.Printf("error shutting down internal HTTP server: %v", err)
		closeErr := s.InternalHTTP.Close()
		if closeErr != nil {
			log.Printf("error closing internal HTTP server: %v", closeErr)
		}
	}
}

func (s *Server) shutdown(ctx context.Context) {
	span, _ := tracing.StartSpanFromContext(ctx, "shutdown")
	defer span.End()

	if s.DB != nil {
		err := s.DB.Close()
		if err != nil {
			log.Printf("error closing database: %v", err)
		}
	}

	if s.Redis != nil {
		err := s.Redis.Close()
		if err != nil {
			log.Printf("error closing redis: %v", err)
		}
	}

	log.Println("client closed")
}
