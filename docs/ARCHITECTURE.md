# Zoff Architecture

This document describes the high-level application architecture across the
`zoff-music/vibes-frontend`, `zoff-music/vibes-backend`, and
`zoff-music/vibes-migrator` repositories. It intentionally focuses on deployed
applications, shared boundaries, data stores, and external providers. The runtime
sections explain how those boundaries support live rooms and background work.

![Zoff application architecture](architecture.svg)

## Components

| Component | Responsibility |
| --- | --- |
| Platform web app | MUSIC and WATCH discovery, search, queues, chat, casting, and synchronized playback |
| Admin web app | Protected room, user, and operational administration |
| Embed web app | Standalone embeddable room player |
| Remote web app | Lightweight controller that pairs with another Zoff screen |
| Cast receiver | Chromecast receiver for synchronized provider playback |
| Mobile app | Native Expo application for iOS and Android phones and tablets |
| TV app | One television product delivered through native Expo Android TV and DOM-based Samsung Tizen renderers |
| Shared frontend packages | Typed API and SSE access, schemas, domain models, state, utilities, renderer-specific UI, SSR serving, styling, and icon exports |
| Vibes backend | Go HTTP API, SSE delivery, remote pairing, playback coordination, and scheduled application-event processing |
| PostgreSQL | Persistent typed rooms, playlist items, playback, participants, generations, staged imports, authorization, and usage counters |
| Redis | Retained event streams for replay, rate limiting, provider-search caching, and short-lived statistics |
| Vibes migrator | Applies PostgreSQL schema changes |
| Media providers | YouTube for MUSIC and WATCH; SoundCloud for MUSIC only |
| Playlist-generation providers | xAI Grok and Google Gemini produce candidate playlists from natural-language prompts |
| Google Cast | Sender SDK, receiver runtime, and media-message transport between Cast-capable clients and the receiver |

## Frontend architecture

All seven applications live in the `zoff-music/vibes-frontend` pnpm workspace.
The web applications use DOM rendering; Platform, Admin, Embed, and Remote use
React Router SSR, while Cast uses a standalone Vite receiver runtime with React
Router data routes. Mobile is
native-only Expo for iOS and Android. TV shares one room/session layer across two
delivery targets: an Expo/React Native renderer for Android TV and a DOM renderer
packaged for Samsung Tizen.

Applications do not import UI or workflows from one another. They compose the
same workspace packages through explicit platform boundaries:

- `@vibes/api` is the only owner of backend REST calls, SSE subscriptions, and
  the generated wire contract.
- `@vibes/models` owns compiled Zod 4 validation schemas and derived domain types.
- `@vibes/shared` owns renderer-neutral hooks, stores, constants, and utilities.
- `@vibes/ui/web`, `@vibes/ui/native`, and `@vibes/ui/shared` keep DOM, React
  Native, and renderer-neutral presentation separate.
- `@vibes/serve` provides the shared SSR server, metrics, and tracing support.
- `@vibes/native-router` provides shared native routing support.
- The Tailwind and iconography packages provide shared styling configuration
  and generated icon exports.

REST capabilities in `@vibes/api` are React-free. Reusable SSE hooks manage
subscription lifecycles, while application state remains app-owned. DOM apps,
including Cast and Tizen, perform REST work in React Router loaders/client
loaders and actions/client actions. Native app hooks compose the same typed
capabilities without importing DOM workflows.

## Backend architecture

| Component | Responsibility |
| --- | --- |
| `cmd/server` and `config` | Process startup and validated environment configuration |
| `server/server.go` | Concrete client initialization, dependency injection, and shutdown |
| `server/router.go` | Versioned routes and route-scoped session, permission, and rate-limit policies |
| `server/internal/handler` | Explicit HTTP request flows and bounded scheduled-event handlers |
| `server/internal/event` | Scheduled-event registration, timers, dispatch, and invocation tracing |
| `vibe` | Domain types, validation, event contracts, and narrow capability interfaces |
| `client/database` | Prepared atomic SQL, row scanning, and mapping to domain data |
| `client/redis` | Event retention/replay, caching, rate-limit state, and statistics |
| Provider clients | External HTTP calls for media metadata, OAuth, and generation |
| `monitoring` | Shared request instrumentation, tracing, and metrics |

There is no service/repository layer. Each handler receives a concrete client
only once through a domain interface containing the capabilities it uses.
Handlers keep their business flow visible; clients implement storage or
provider-specific operations and do not call other clients. The v1 and v2 room
event handlers have separate implementations in `events.go` and `eventsv2.go`;
the playlist-item v3 stream lives in `roomv3.go`.

Feature files group their HTTP flows and scheduled work together: `admin.go`,
`listener.go`, `playback.go`, `playlist.go`, and `generate.go`. Client feature
files keep the matching prepared statements, operations, scans, and mappings
together. Admin search-usage reads belong in `client/database/admin.go`;
listener-usage reads belong in `client/database/listener.go`. API versions retain
their explicit files and wire contracts, rather than sharing a version-switching
handler.

Database methods use context-aware prepared statements and bounded timeouts.
Row scanning is separate from `toXxx` domain mapping. Atomic statements and
guarded worker claims coordinate multiple backend instances without
Go-managed transactions.

## Runtime data flow

Frontend clients issue typed HTTPS requests through `@vibes/api` and subscribe
to room or remote SSE streams for live updates. The backend coordinates room and
playback state, persists durable state in PostgreSQL, and uses Redis for retained
delivery events and short-lived operational data. Scheduled app events run
inside the Go process; Redis is not a separate worker/job runner. Their handlers
claim persistent work, advance playback, import or generate playlists, refresh
provider state, and perform maintenance.

Media-provider integration crosses two boundaries: the backend calls provider
APIs for search, metadata, playlists, and authorization where supported, while
playback applications use the providers' official players or SDKs. Cast-capable
clients communicate with the Cast receiver through Google Cast media messages;
the receiver also uses the same backend API and SSE contracts as the other
applications.

## MUSIC and WATCH rooms

Room type and playback mode are independent. PostgreSQL stores the immutable
`rooms.room_type` enum as `MUSIC` or `WATCH`; existing rooms were backfilled to
`MUSIC`. Both types use the same playlist-item storage and event infrastructure.

| Behaviour | MUSIC | WATCH |
| --- | --- | --- |
| Providers | YouTube and SoundCloud when configured and enabled | YouTube only |
| YouTube search and imports | Music-category videos | Videos across categories |
| Generation | Music-oriented prompt and candidates | Viewing-oriented prompt and candidates |
| Discovery and totals | MUSIC rooms, queued items, listeners | WATCH rooms, queued items, watchers |
| Playback authority | Server or host mode | Server or host mode |

Room-scoped search loads the stored type before choosing provider policy. Search
cache keys include the type, and additions/imports revalidate provider metadata
against the destination room. Both types reject live, made-for-kids, and
non-embeddable YouTube videos; age and regional restrictions remain represented
in provider metadata. WATCH does not apply a music-category or video-length
restriction. Public lists and community totals accept a room-type filter so the
two experiences do not mix results. Host identity and handovers travel through
room events and system activity, independently of room-administrator status.

## State and event delivery

| Channel | Responsibility |
| --- | --- |
| `/api/v1` REST | Retained compatibility contracts, plus shared sessions, chat, generation requests, and provider configuration |
| `/api/v2/rooms` REST | Typed rooms, playlist items, settings, playback, failures, and imports |
| `/api/v2/rooms/{id}/search/{provider}` | Search using the stored room type and enabled providers |
| `/api/v1/rooms/{id}/events` | Compatibility room stream with full queue updates |
| `/api/v2/rooms/{id}/events` | Initial room/queue state followed by incremental song and playback events |
| `/api/v3/rooms/{id}/events` | Playlist-item snapshots and incremental room, queue, host, and playback events |
| `/api/v3/rooms/public` | Paginated public-room browsing filtered by MUSIC or WATCH |
| `/api/v2/stats` | Community counts filtered by room type |
| `/api/v1/rooms/{id}/messages` | POST chat messages and GET the chat/activity SSE stream |
| `/api/v2/remotes/{id}/events` | Paired-player state and targeted remote commands using playlist-item contracts |

Room event IDs support reconnect replay. A valid retained cursor resumes after
the last event; a new connection or expired cursor requires a fresh snapshot.
The v2 and v3 queues send additions, updates, removals, and position changes rather
than a full playlist for each mutation. Heartbeats keep streams and presence
alive without repeatedly transmitting the queue.

Redis replay is bounded by `ROOM_EVENT_REPLAY_MAX_EVENTS` and
`ROOM_EVENT_REPLAY_MAX_AGE` (defaults: 100 events and 15 minutes). It is a recovery
window, not a permanent event archive. Chat and room activity share this live
delivery infrastructure; PostgreSQL stores usage counters, not an unbounded
message transcript. Clients also bound their rendered message history. A
per-device chat preference hides chat locally without changing the room.

## Playlist imports and background work

Playlist imports validate room permissions and provider tracks before staging
items. Each staged item is inserted separately with its complete track data and
position, rather than relying on parallel arrays. Only a complete, validated
import is published for workers to claim. Staging has an overall two-minute
request budget; each database operation retains its own bounded timeout.

The scheduled import handler processes one item per invocation at a 100 ms
interval. Queue changes are broadcast as normal room events, and importing into
an empty room initializes playback. Failed staging is cleaned up; maintenance
also handles abandoned imports. Generation, metadata refresh, playback
progression, and cleanup are separate scheduled flows with bounded work and
concurrency-safe database claims where needed.

YouTube provider lookups exclude videos marked as made for kids from searches,
direct track lookups, playlist imports, and generated selections. Playlist
responses report separate counts for made-for-kids and embedding exclusions.
YouTube additions require recently verified provider metadata in the shared
cache; expired previews must be loaded again instead of trusting client data.

Provider metadata refresh includes the currently selected playlist item, even when
paused, and broadcasts updated queue and playback metadata. A separate expiry
worker removes unrefreshed YouTube metadata after 25 days, independently of
provider availability or quota. It claims one room at a time and atomically
removes stale items and votes, clearing playback if its selected item expires.
The cutoff leaves headroom for three-day caches, one-day import staging, and
short-lived event replay before the 30-day metadata limit. Import cleanup also
expires unfinished imports after one day rather than retaining their items.

## Sessions and permissions

Listeners and watchers use signed anonymous sessions, not required user accounts. Room
administrator authentication is scoped to that room; creating a room with an
administrator password also authenticates its creator. Global administration
has a separate protected session and permission boundary.

Room settings govern adding, skipping, duplicate tracks, queue behavior, and
public discovery. Server mode advances playback automatically. Host mode gives
the host control, with administrator permissions handled explicitly. The
backend validates every mutation independently of which controls a client
shows. Chat text, display names, and newly created room names have server-side
length limits as well as frontend validation.

Remote pairing identifies an owner/player and a controller. Targeted player
commands remain distinct from room-wide state updates so a controller does not
become another listener or echo its own commands. Cast uses room-scoped tokens
instead of copying browser cookies to the receiver.

## Schema and release boundaries

The migrator repository owns numbered SQL migrations and generated database
documentation. The backend consumes that schema but does not migrate it at
startup. Schema-dependent changes are released in compatible order: migrator,
backend, then frontend consumers. Old and new application instances may overlap
during a rolling deployment.

The backend ships as a static Go binary in a scratch image. Frontend SSR apps
use the shared Node serving package; Cast and Tizen have separate DOM delivery
targets. Mobile and Android TV use local native builds and their Expo/EAS release
configuration, independent of web deployment. Documentation-only updates do not
require a native app release.

## Process lifecycle

Startup initializes clients and statements before accepting requests and
starting scheduled work. Redis remains required even when rate limiting is off.
Shutdown cancels request and worker contexts, drains HTTP servers, and waits
for active goroutines before closing database, Redis, and provider clients.
This ordering prevents SSE readers and heartbeat handlers from using closed
dependencies during deployments.

Detailed user and system interactions are documented in [Application flows](FLOWS.md).
