# 2oFF Architecture

This document describes the high-level application architecture. It intentionally
focuses on deployed applications, data stores, and external providers rather than
implementation-level worker details.

![2oFF application architecture](architecture.svg)

## Components

| Component | Responsibility |
| --- | --- |
| Platform | Main listener interface for rooms, search, playlists, and playback |
| Admin | Operational room administration |
| Embed | Embeddable room player |
| Cast receiver | Chromecast playback application |
| Vibes backend | Go API, SSE event delivery, playback coordination, and background processing |
| PostgreSQL | Persistent rooms, songs, playback, participants, generations, and authorization data |
| Redis | Rate limiting, normalized provider-search caching, and short-lived statistics |
| Vibes migrator | Applies PostgreSQL schema changes |
| Music providers | Search, metadata, and provider playback |
| Playlist-generation providers | Produce candidate playlists from natural-language prompts |

Detailed user and system interactions are documented in [Application flows](FLOWS.md).
