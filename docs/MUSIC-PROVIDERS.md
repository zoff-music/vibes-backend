# Media Providers

Zoff supports YouTube and SoundCloud for search, metadata, playlists, and
synchronized playback. The backend returns provider metadata; it never hosts,
downloads, extracts, proxies, transcodes, records, or redistributes media.
Playback remains in each provider's official maintained player in the frontend.
MUSIC rooms support both providers. WATCH rooms support YouTube only, enforced
in room creation, settings, search, additions and imports.

## YouTube

- Search: `GET /api/v2/rooms/{id}/search/youtube?q=query`
- Details: `GET /api/v2/youtube/videos/{id}`
- Playlists: `GET /api/v2/youtube/playlists/{id}?roomType=MUSIC` or `WATCH`
- Configuration: `YOUTUBE_API_KEY`
- Playback: official YouTube IFrame Player API

## SoundCloud

- Search: `GET /api/v2/rooms/{id}/search/soundcloud?q=query` (MUSIC only)
- Item resolution: `GET /api/v2/soundcloud/items?url=...`
- Playlist resolution: `GET /api/v2/soundcloud/playlists?url=...`
- Configuration: `SOUNDCLOUD_CLIENT_ID` and `SOUNDCLOUD_CLIENT_SECRET`
- Playback: official SoundCloud embedded widget

## Provider contract

Current provider responses use `ProviderItem` and `ProviderPlaylist`, with
`publisher` metadata. Legacy v1 endpoints retain `MusicTrack` and `MusicPlaylist`
through explicit mapping. `GET /api/v1/providers` reports only providers with the required server
credentials configured. Room `enabledSources` values are limited to the same
provider set and the room type. MUSIC YouTube searches/imports accept the music
category; WATCH is not category-limited. Both reject live, made-for-kids and
non-embeddable videos, while retaining age/country restriction information.

Provider credentials belong in deployment secrets and must never be committed.
