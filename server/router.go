package server

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zoff-music/vibes-backend/server/internal/handler"
	"github.com/zoff-music/vibes-backend/server/internal/middleware"
	_ "github.com/zoff-music/vibes-backend/swaggerdocs"
	"github.com/zoff-music/vibes-backend/vibe"
)

// setupRoutes - the root route function.
func (s *Server) setupRoutes() {
	// Analytics is cookie-free and must not create an application session.
	analytics := s.Router.NewRoute().Subrouter()
	analytics.HandleFunc(v1API+"/usage-events", handler.CreateAnalyticsEvent(s.Plausible, s.Config.PlausibleDomain)).Methods(http.MethodPost, http.MethodOptions).Name("CreateAnalyticsEvent")
	s.addTracingAndMetrics(analytics)
	s.addRateLimitMiddleware(analytics)
	apiV1 := s.Router.PathPrefix(v1API).Subrouter()
	apiV2 := s.Router.PathPrefix(v2API).Subrouter()
	apiV3 := s.Router.PathPrefix(v3API).Subrouter()
	apiV3.HandleFunc("/rooms/public", handler.GetPublicRoomsV3(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPublicRoomsV3")
	apiV3.HandleFunc("/rooms/{id}/events", handler.RoomEventsV3(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RoomEventsV3")
	apiV3.HandleFunc("/rooms/{id}/items", handler.GetItemsV3(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetItemsV3")
	s.Router.Handle(swaggerAPI, http.RedirectHandler(swaggerAPI+"/", http.StatusPermanentRedirect)).Methods(http.MethodGet)

	docs := s.Router.PathPrefix(swaggerAPI + "/").Subrouter()

	if s.Config.AdminPasswordPepper != "" {
		docs.HandleFunc("/admin.json", handler.AdminDocumentation()).Methods(http.MethodGet).Name("AdminDocumentation")
		s.addSessionMiddleware(docs)
		s.addAdminMiddleware(docs)
	}

	docs.PathPrefix("/").Handler(handler.Documentation(s.Config.DocumentationDirectory)).Methods(http.MethodGet)

	// Room routes
	apiV2.HandleFunc("/rooms", handler.CreateRoomV2(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoomV2")
	apiV1.HandleFunc("/rooms/reservations", handler.ReserveRoomName(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("ReserveRoomName")
	apiV1.HandleFunc("/rooms/suggestions", handler.SuggestRoomName(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SuggestRoomName")
	apiV2.HandleFunc("/rooms/generation", handler.CreateGeneratedRoomV2(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("CreateGeneratedRoomV2")
	apiV1.HandleFunc("/generations", handler.CreateGeneration(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateGeneration")
	apiV1.HandleFunc("/generations/{id}", handler.GetGeneration(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetGeneration")
	apiV1.HandleFunc("/rooms/{id}/generations", handler.CreateRoomGeneration(s.DB, s.Config.RoomGenerationMaxExistingPlaylistItems)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoomGeneration")
	apiV1.HandleFunc("/rooms/{id}", handler.RoomExists(s.DB)).Methods(http.MethodHead).Name("RoomExists")
	apiV2.HandleFunc("/rooms/{id}", handler.GetRoomV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRoomV2")
	apiV2.HandleFunc("/rooms/{id}/settings", handler.UpdateRoomSettingsV2(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRoomSettingsV2")
	apiV2.HandleFunc("/rooms/{id}/skips", handler.SkipPlaylistItem(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("SkipPlaylistItem")
	apiV2.HandleFunc("/rooms/{id}/states", handler.GetPlaybackStateV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPlaybackStateV2")
	apiV2.HandleFunc("/rooms/{id}/states", handler.UpdatePlaybackStateV2(s.DB, s.Redis)).Methods(http.MethodPut, http.MethodOptions).Name("UpdatePlaybackStateV2")
	apiV2.HandleFunc("/rooms/{id}/failures", handler.ReportPlaybackFailureV2(s.DB, s.YouTube, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("ReportPlaybackFailureV2")
	apiV2.HandleFunc("/rooms/{id}/sessions", handler.CreateSessionV2(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("CreateSessionV2")
	apiV2.HandleFunc("/rooms/{id}/sessions", handler.DeleteRoomAdminSessionV2(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("DeleteRoomAdminSessionV2")
	apiV1.HandleFunc("/sessions", handler.GetSessionProfile(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetSessionProfile")
	apiV1.HandleFunc("/sessions", handler.UpdateSessionProfile(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateSessionProfile")

	// Playlist item routes.
	apiV3.HandleFunc("/rooms/{id}/items", handler.AddItemV3(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("AddItemV3")
	apiV3.HandleFunc("/rooms/{id}/items/{itemId}", handler.RemoveItemV3(s.DB, s.Redis)).Methods(http.MethodDelete, http.MethodOptions).Name("RemoveItemV3")
	apiV3.HandleFunc("/rooms/{id}/items/{itemId}", handler.VoteItemV3(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("VoteItemV3")
	apiV2.HandleFunc("/rooms/{id}/playlists", handler.AddPlaylistV2(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("AddPlaylistV2")

	// SSE route
	apiV1.HandleFunc("/rooms/{id}/messages", handler.CreateMessages(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("CreateMessages")
	apiV1.HandleFunc("/rooms/{id}/messages", handler.Messages(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("Messages")

	// YouTube routes
	apiV2.HandleFunc("/rooms/{id}/search/{provider:youtube}", handler.SearchYouTubeV2(s.YouTube, s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SearchYouTubeV2")
	apiV2.HandleFunc("/rooms/{id}/search/{provider:soundcloud}", handler.SearchSoundCloudV2(s.SoundCloud, s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SearchSoundCloudV2")
	apiV2.HandleFunc("/youtube/videos/{id}", handler.GetYouTubeItemV2(s.YouTube, s.Redis, vibe.SourceTypeYouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubeItemV2")
	apiV2.HandleFunc("/youtube/playlists/{id}", handler.GetYouTubePlaylistV2(s.YouTube, s.Redis, vibe.SourceTypeYouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubePlaylistV2")

	// SoundCloud routes
	apiV2.HandleFunc("/soundcloud/items", handler.ResolveSoundCloudItemV2(s.SoundCloud, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("ResolveSoundCloudItemV2")
	apiV2.HandleFunc("/soundcloud/items/{id}", handler.GetSoundCloudItemV2(s.SoundCloud, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudItemV2")
	apiV2.HandleFunc("/soundcloud/playlists", handler.ResolveSoundCloudPlaylistV2(s.SoundCloud, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("ResolveSoundCloudPlaylistV2")

	apiV1.HandleFunc("/tokens/soundcloud", handler.GetToken(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudToken")
	apiV1.HandleFunc("/tokens/youtube", handler.GetToken(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubeToken")

	// Authorization routes
	apiV1.HandleFunc("/authorizations/soundcloud", handler.Authorize(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	apiV1.HandleFunc("/authorizations/youtube", handler.Authorize(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	// Callbacks
	apiV1.HandleFunc("/callbacks/soundcloud", handler.OAuthCallback(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("SoundCloudCallback")
	apiV1.HandleFunc("/callbacks/youtube", handler.OAuthCallback(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("YouTubeCallback")

	// Config routes
	apiV1.HandleFunc("/providers", handler.GetProviders(s.Config)).Methods(http.MethodGet, http.MethodOptions).Name("GetProviders")

	// Stats routes
	apiV2.HandleFunc("/stats", handler.GetStatsV2(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("GetStatsV2")

	// Cast token endpoint (cookie-auth only)
	apiV1.HandleFunc("/tokens/casting", handler.CreateCastingToken(s.DB, s.Config.CastTokenSecret)).Methods(http.MethodPost, http.MethodOptions).Name("CreateCastingToken")

	// Remote control routes
	apiV2.HandleFunc("/remotes", handler.CreateRemoteControlV2(s.DB, s.Redis, s.Config.CookieSecret, s.Config.RemotePairingTTL)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRemoteControlV2")
	apiV2.HandleFunc("/remotes", handler.GetOwnedRemoteControlV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetOwnedRemoteControlV2")
	apiV2.HandleFunc("/remotes/{id}", handler.GetRemoteControlV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRemoteControlV2")
	apiV2.HandleFunc("/remotes/{id}", handler.UpdateRemoteControlV2(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRemoteControlV2")
	apiV1.HandleFunc("/remotes/{id}", handler.DeleteRemoteControl(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("DeleteRemoteControl")
	apiV2.HandleFunc("/remotes/{id}/sessions", handler.PairRemoteControlV2(s.DB, s.Redis, s.Config.CookieSecret)).Methods(http.MethodPost, http.MethodOptions).Name("PairRemoteControlV2")
	apiV2.HandleFunc("/remotes/{id}/events", handler.RemoteEventsV2(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RemoteEventsV2")

	// Admin routes

	if s.Config.AdminPasswordPepper != "" {
		apiV1.HandleFunc("/admin/sessions", handler.AdminLogin(s.DB, s.Config.AdminPasswordPepper, s.Config.CookieSecret)).Methods(http.MethodPost, http.MethodOptions).Name("AdminLogin")
		apiV1.HandleFunc("/admin/sessions", handler.AdminSession()).Methods(http.MethodGet, http.MethodOptions).Name("AdminSession")
		apiV1.HandleFunc("/admin/sessions", handler.AdminLogout()).Methods(http.MethodDelete, http.MethodOptions).Name("AdminLogout")
		apiV1.HandleFunc("/admin/users", handler.AdminUsers(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminUsers")
		apiV1.HandleFunc("/admin/users", handler.AdminCreateUser(s.DB, s.Config.AdminPasswordPepper)).Methods(http.MethodPost, http.MethodOptions).Name("AdminCreateUser")
		apiV1.HandleFunc("/admin/users/{id}", handler.AdminUpdateUser(s.DB, s.Config.AdminPasswordPepper)).Methods(http.MethodPatch, http.MethodOptions).Name("AdminUpdateUser")
		apiV1.HandleFunc("/admin/users/{id}", handler.AdminDeleteUser(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("AdminDeleteUser")
		apiV2.HandleFunc("/admin/rooms", handler.AdminRoomsV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminRoomsV2")
		apiV1.HandleFunc("/admin/messages/usage", handler.AdminMessageUsage(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminMessageUsage")
		apiV1.HandleFunc("/admin/searches/usage", handler.AdminSearchUsage(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("AdminSearchUsage")
		apiV1.HandleFunc("/admin/listeners/usage", handler.AdminListenerUsage(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("AdminListenerUsage")
		apiV2.HandleFunc("/admin/rooms/{id}", handler.AdminUpdateRoomV2(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("AdminUpdateRoomV2")
		apiV2.HandleFunc("/admin/rooms/{id}", handler.AdminDeleteRoomV2(s.DB, s.Redis)).Methods(http.MethodDelete, http.MethodOptions).Name("AdminDeleteRoomV2")
		apiV3.HandleFunc("/admin/events", handler.AdminEventsV3(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminEventsV3")
	}

	s.addTracingAndMetrics(apiV1, apiV2, apiV3)
	s.addSessionMiddleware(apiV1, apiV2, apiV3)
	s.addRateLimitMiddleware(apiV1, apiV2, apiV3)
	s.addPermissionMiddleware(apiV1, apiV2, apiV3)

	if s.Config.AdminPasswordPepper != "" {
		s.addAdminMiddleware(apiV1, apiV2, apiV3)
	}
	s.addCORSMiddleware(s.Router)
}

func (s *Server) addRateLimitMiddleware(routers ...*mux.Router) {
	rm := middleware.RateLimitMiddleware{
		Enabled: s.Config.RateLimitEnabled,
		RequiredRoutes: map[string]bool{
			"CreateAnalyticsEvent": true,
			"AdminLogin":           true,
			"CreateMessages":       true,
		},
		Checker: s.Redis,
		Policies: map[string]vibe.RateLimitPolicy{
			"CreateAnalyticsEvent": {Rate: time.Minute, Limit: 120, IPLimit: 1200},
			"CreateRoomV2":         {Bucket: "CreateRoom", Rate: time.Minute, Limit: 10},
			"ReserveRoomName": {
				Bucket:  "RoomNameReservations",
				Rate:    time.Minute,
				Limit:   12,
				IPLimit: 30,
			},
			"SuggestRoomName": {
				Bucket:  "RoomNameReservations",
				Rate:    time.Minute,
				Limit:   12,
				IPLimit: 30,
			},
			"RoomExists":                  {Rate: time.Minute, Limit: 60},
			"GetRoomV2":                   {Bucket: "GetRoom", Rate: time.Minute, Limit: 120},
			"GetPublicRoomsV3":            {Bucket: "GetPublicRoomsV2", Rate: time.Minute, Limit: 120},
			"UpdateRoomSettingsV2":        {Bucket: "UpdateRoomSettings", Rate: time.Minute, Limit: 30},
			"SkipPlaylistItem":            {Bucket: "SkipSong", Rate: time.Minute, Limit: 60},
			"GetPlaybackStateV2":          {Bucket: "GetPlaybackState", Rate: time.Minute, Limit: 240},
			"UpdatePlaybackStateV2":       {Bucket: "UpdatePlaybackState", Rate: time.Minute, Limit: 240},
			"ReportPlaybackFailureV2":     {Bucket: "ReportPlaybackFailure", Rate: time.Minute, Limit: 30},
			"CreateSessionV2":             {Bucket: "CreateSession", Rate: time.Minute, Limit: 30},
			"GetItemsV3":                  {Bucket: "GetSongs", Rate: time.Minute, Limit: 120},
			"AddItemV3":                   {Bucket: "AddSong", Rate: time.Minute, Limit: 60},
			"AddPlaylistV2":               {Bucket: "AddPlaylist", Rate: time.Minute, Limit: 6},
			"RemoveItemV3":                {Bucket: "RemoveSong", Rate: time.Minute, Limit: 60},
			"VoteItemV3":                  {Bucket: "VoteSong", Rate: time.Minute, Limit: 120},
			"CreateMessages":              {Rate: time.Second, Limit: 1},
			"Messages":                    {Rate: time.Minute, Limit: 30},
			"RoomEventsV3":                {Bucket: "RoomEventsV2", Rate: time.Minute, Limit: 30},
			"GetYouTubeItemV2":            {Bucket: "GetMusicTrack", Rate: time.Minute, Limit: 120},
			"GetYouTubePlaylistV2":        {Bucket: "GetYouTubePlaylist", Rate: time.Minute, Limit: 10},
			"GetSoundCloudItemV2":         {Bucket: "GetSoundCloudTrack", Rate: time.Minute, Limit: 120},
			"ResolveSoundCloudItemV2":     {Bucket: "ResolveSoundCloudTrack", Rate: time.Second, Limit: 3, IPLimit: 30},
			"ResolveSoundCloudPlaylistV2": {Bucket: "ResolveSoundCloudPlaylist", Rate: time.Minute, Limit: 10},
			"SearchYouTubeV2": {
				Bucket:  "YouTubeSearch",
				Rate:    time.Second,
				Limit:   1,
				IPLimit: 3,
			},
			"SearchSoundCloudV2": {
				Bucket:  "SearchSoundCloud",
				Rate:    time.Second,
				Limit:   1,
				IPLimit: 3,
			},
			"GetSoundCloudToken": {Rate: time.Minute, Limit: 60},
			"GetYouTubeToken":    {Rate: time.Minute, Limit: 60},
			"CreateGeneratedRoomV2": {
				Bucket:      "RoomGeneration",
				Rate:        10 * time.Minute,
				Limit:       1,
				IPLimit:     2,
				GlobalRate:  time.Minute,
				GlobalLimit: 1,
			},
			"CreateGeneration": {
				Bucket:      "RoomGeneration",
				Rate:        10 * time.Minute,
				Limit:       1,
				IPLimit:     2,
				GlobalRate:  time.Minute,
				GlobalLimit: 1,
			},
			"GetGeneration": {Rate: time.Minute, Limit: 120},
			"CreateRoomGeneration": {
				Bucket:      "RoomGeneration",
				Rate:        10 * time.Minute,
				Limit:       1,
				IPLimit:     2,
				GlobalRate:  time.Minute,
				GlobalLimit: 1,
			},
			"Authorize":               {Rate: 10 * time.Minute, Limit: 20},
			"SoundCloudCallback":      {Rate: 10 * time.Minute, Limit: 30},
			"YouTubeCallback":         {Rate: 10 * time.Minute, Limit: 30},
			"GetProviders":            {Rate: time.Minute, Limit: 120},
			"GetStatsV2":              {Bucket: "GetStats", Rate: time.Minute, Limit: 600},
			"CreateCastingToken":      {Rate: time.Minute, Limit: 30},
			"CreateRemoteControlV2":   {Bucket: "CreateRemoteControl", Rate: time.Minute, Limit: 10},
			"GetOwnedRemoteControlV2": {Bucket: "GetOwnedRemoteControl", Rate: time.Minute, Limit: 120},
			"GetRemoteControlV2":      {Bucket: "GetRemoteControl", Rate: time.Minute, Limit: 120},
			"UpdateRemoteControlV2":   {Bucket: "UpdateRemoteControl", Rate: time.Minute, Limit: 180},
			"DeleteRemoteControl":     {Rate: time.Minute, Limit: 10},
			"PairRemoteControlV2": {
				Bucket:  "PairRemoteControl",
				Rate:    10 * time.Minute,
				Limit:   5,
				IPLimit: 10,
			},
			"RemoteEventsV2":     {Bucket: "RemoteEvents", Rate: time.Minute, Limit: 30},
			"AdminLogin":         {Rate: 10 * time.Minute, Limit: 5},
			"AdminSession":       {Rate: time.Minute, Limit: 120},
			"AdminLogout":        {Rate: time.Minute, Limit: 20},
			"AdminUsers":         {Rate: time.Minute, Limit: 60},
			"AdminCreateUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminUpdateUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminDeleteUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminRoomsV2":       {Bucket: "AdminRooms", Rate: time.Minute, Limit: 120},
			"AdminMessageUsage":  {Rate: time.Minute, Limit: 120},
			"AdminSearchUsage":   {Rate: time.Minute, Limit: 120},
			"AdminListenerUsage": {Rate: time.Minute, Limit: 120},
			"AdminUpdateRoomV2":  {Bucket: "AdminUpdateRoom", Rate: time.Minute, Limit: 30},
			"AdminDeleteRoomV2":  {Bucket: "AdminDeleteRoom", Rate: time.Minute, Limit: 30},
			"AdminEventsV3":      {Bucket: "AdminEvents", Rate: time.Minute, Limit: 20},
		},
	}

	for _, r := range routers {
		r.Use(rm.Middleware)
	}
}

func (s *Server) setupInternalRoutes() {
	s.InternalRouter.HandleFunc("/_healthz", handler.Healthz).Methods(http.MethodGet).Name("Health")
	s.InternalRouter.Handle("/metrics", promhttp.Handler()).Name("Metrics")
}

func (s *Server) addSessionMiddleware(routers ...*mux.Router) {
	sm := middleware.SessionMiddleware{
		CastRouteNames: map[string]bool{
			"ReportPlaybackFailureV2": true,
			"RoomEventsV3":            true,
			"GetPlaybackStateV2":      true,
			"GetRoomV2":               true,
			"GetProviders":            true,
			"GetItemsV3":              true,
		},
		RemoteRouteNames: map[string]bool{
			"AddPlaylistV2":               true,
			"RoomEventsV3":                true,
			"SkipPlaylistItem":            true,
			"GetPlaybackStateV2":          true,
			"UpdatePlaybackStateV2":       true,
			"AddItemV3":                   true,
			"RemoveItemV3":                true,
			"VoteItemV3":                  true,
			"GetRoomV2":                   true,
			"UpdateRoomSettingsV2":        true,
			"GetRemoteControlV2":          true,
			"UpdateRemoteControlV2":       true,
			"RemoteEventsV2":              true,
			"CreateSessionV2":             true,
			"GetItemsV3":                  true,
			"SearchYouTubeV2":             true,
			"SearchSoundCloudV2":          true,
			"GetYouTubeItemV2":            true,
			"GetYouTubePlaylistV2":        true,
			"ResolveSoundCloudItemV2":     true,
			"GetSoundCloudItemV2":         true,
			"ResolveSoundCloudPlaylistV2": true,
			"GetProviders":                true,
		},
		RemoteRoomRouteNames: map[string]bool{
			"AddPlaylistV2":         true,
			"RoomEventsV3":          true,
			"SkipPlaylistItem":      true,
			"GetPlaybackStateV2":    true,
			"UpdatePlaybackStateV2": true,
			"AddItemV3":             true,
			"RemoveItemV3":          true,
			"VoteItemV3":            true,
			"GetRoomV2":             true,
			"UpdateRoomSettingsV2":  true,
			"SearchYouTubeV2":       true,
			"SearchSoundCloudV2":    true,
			"CreateSessionV2":       true,
			"GetItemsV3":            true,
		},
		Secret:                     s.Config.CookieSecret,
		CookieMaxAge:               s.Config.SessionCookieMaxAge,
		CastTokenSecret:            s.Config.CastTokenSecret,
		EmbedBasePath:              s.Config.EmbedBasePath,
		RemoteControlAuthenticator: s.DB,
	}
	for _, r := range routers {
		r.Use(sm.Middleware)
	}
}

// addTracingAndMetrics - Adds tracing and metrics to a router.
func (s *Server) addTracingAndMetrics(routers ...*mux.Router) {
	tm := middleware.TraceMiddleware{
		ExemptRoutes: map[string]bool{
			"RoomEventsV3":   true,
			"Messages":       true,
			"RemoteEventsV2": true,
			"AdminEventsV3":  true,
		},
	}
	for _, r := range routers {
		r.Use(tm.Middleware)
		r.Use(middleware.MetricsMiddleware)
	}
}

func (s *Server) addCORSMiddleware(routers ...*mux.Router) {
	cm := &middleware.CORSMiddleware{
		AllowedOriginsCSV: s.Config.CORSAllowedOrigins,
	}
	for _, r := range routers {
		r.Use(cm.Middleware)
	}
}

func (s *Server) addPermissionMiddleware(routers ...*mux.Router) {
	am := middleware.PermissionMiddleware{
		DB: s.DB,
		ProtectedRoutes: map[string]bool{
			"CreateRoomGeneration": true,
			"UpdateRoomSettingsV2": true,
		},
	}

	for _, r := range routers {
		r.Use(am.Middleware)
	}
}

func (s *Server) addAdminMiddleware(routers ...*mux.Router) {
	am := middleware.AdminMiddleware{
		DB:           s.DB,
		CookieSecret: s.Config.CookieSecret,
		OptionalRoutes: map[string]bool{
			"CreateMessages":       true,
			"CreateSessionV2":      true,
			"UpdateSessionProfile": true,
			"UpdateRoomSettingsV2": true,
			"AddItemV3":            true,
			"RemoveItemV3":         true,
			"VoteItemV3":           true,
			"SkipPlaylistItem":     true,
		},
		ProtectedRoutes: map[string]bool{
			"AdminSession":       true,
			"AdminDocumentation": true,
			"AdminUsers":         true,
			"AdminCreateUser":    true,
			"AdminUpdateUser":    true,
			"AdminDeleteUser":    true,
			"AdminRoomsV2":       true,
			"AdminMessageUsage":  true,
			"AdminSearchUsage":   true,
			"AdminListenerUsage": true,
			"AdminUpdateRoomV2":  true,
			"AdminDeleteRoomV2":  true,
			"AdminEventsV3":      true,
		},
	}

	for _, r := range routers {
		r.Use(am.Middleware)
	}
}

const v1API string = "/api/v1"
const v2API string = "/api/v2"
const v3API string = "/api/v3"
const swaggerAPI string = "/api/swagger"
