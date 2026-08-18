# Music Providers

Zoff supports YouTube and SoundCloud for search, metadata, playlists, and
synchronized playback. The backend returns provider metadata; it never hosts,
downloads, extracts, proxies, transcodes, records, or redistributes media.
Playback remains in each provider's official maintained player in the frontend.

## YouTube

- Search: `GET /api/v1/youtube/search?q=query`
- Details: `GET /api/v1/youtube/videos/{id}`
- Playlists: `GET /api/v1/youtube/playlists/{id}`
- Configuration: `YOUTUBE_API_KEY`
- Playback: official YouTube IFrame Player API

## SoundCloud

- Search: `GET /api/v1/soundcloud/search?q=query`
- Track resolution: `GET /api/v1/soundcloud/tracks?url=...`
- Playlist resolution: `GET /api/v1/soundcloud/playlists?url=...`
- Configuration: `SOUNDCLOUD_CLIENT_ID` and `SOUNDCLOUD_CLIENT_SECRET`
- Playback: official SoundCloud embedded widget

## Provider contract

Provider responses use the shared `MusicTrack` and `MusicPlaylist` domain
types. `GET /api/v1/providers` reports only providers with the required server
credentials configured. Room `enabledSources` values are limited to the same
provider set.

Provider credentials belong in deployment secrets and must never be committed.
