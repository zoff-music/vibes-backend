# Zoff Architecture

This document describes the high-level application architecture across the
`zoff-music/vibes-frontend`, `zoff-music/vibes-backend`, and
`zoff-music/vibes-migrator` repositories. It intentionally focuses on deployed
applications, shared boundaries, data stores, and external providers rather than
implementation-level worker details.

![Zoff application architecture](architecture.svg)

## Components

| Component | Responsibility |
| --- | --- |
| Platform web app | Primary listener interface for rooms, search, queues, casting, and synchronized playback |
| Admin web app | Protected room, user, and operational administration |
| Embed web app | Standalone embeddable room player |
| Remote web app | Lightweight controller that pairs with another Zoff screen |
| Cast receiver | Chromecast receiver for synchronized provider playback |
| Mobile app | Native Expo application for iOS and Android phones and tablets |
| TV app | One television product delivered through native Expo Android TV and DOM-based Samsung Tizen renderers |
| Shared frontend packages | Typed API and SSE access, schemas, domain models, state, utilities, renderer-specific UI, SSR serving, styling, and icon exports |
| Vibes backend | Go HTTP API, SSE delivery, remote pairing, playback coordination, and durable application-event processing |
| PostgreSQL | Persistent rooms, songs, playback, participants, generations, and authorization data |
| Redis | Durable application-event streams, rate limiting, provider-search caching, and short-lived statistics |
| Vibes migrator | Applies PostgreSQL schema changes |
| Music providers | YouTube, Spotify, and SoundCloud search, metadata, authorization, and playback surfaces |
| Playlist-generation providers | xAI Grok and Google Gemini produce candidate playlists from natural-language prompts |
| Google Cast | Sender SDK, receiver runtime, and media-message transport between Cast-capable clients and the receiver |

## Frontend architecture

All seven applications live in the `zoff-music/vibes-frontend` pnpm workspace.
The web applications use DOM rendering; Platform, Admin, Embed, and Remote use
React Router SSR, while Cast uses a standalone Vite receiver runtime. Mobile is
native-only Expo for iOS and Android. TV shares one room/session layer across two
delivery targets: an Expo/React Native renderer for Android TV and a DOM renderer
packaged for Samsung Tizen.

Applications do not import UI or workflows from one another. They compose the
same workspace packages through explicit platform boundaries:

- `@vibes/api` is the only owner of backend REST calls, SSE subscriptions, and
  the generated wire contract.
- `@vibes/models` owns shared validation schemas and derived domain types.
- `@vibes/shared` owns renderer-neutral hooks, stores, constants, and utilities.
- `@vibes/ui/web`, `@vibes/ui/native`, and `@vibes/ui/shared` keep DOM, React
  Native, and renderer-neutral presentation separate.
- `@vibes/serve` provides the shared SSR server, metrics, and tracing support.
- The Tailwind and iconography packages provide shared styling configuration
  and generated icon exports.

## Runtime data flow

Frontend clients issue typed HTTPS requests through `@vibes/api` and subscribe
to room or remote SSE streams for live updates. The backend coordinates room and
playback state, persists durable state in PostgreSQL, and uses Redis for durable
application events and short-lived operational data. Background event handlers
advance playback, generate playlists, refresh provider state, and perform
maintenance.

Music-provider integration crosses two boundaries: the backend calls provider
APIs for search, metadata, playlists, and authorization where supported, while
playback applications use the providers' official players or SDKs. Cast-capable
clients communicate with the Cast receiver through Google Cast media messages;
the receiver also uses the same backend API and SSE contracts as the other
applications.

Detailed user and system interactions are documented in [Application flows](FLOWS.md).
