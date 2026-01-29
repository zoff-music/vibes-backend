import type { Room, Song } from '@vibez/models';
import {
  PlayerControls,
  SoundCloudPlayer,
  SpotifyPlayer,
  VideoPlayer,
} from '@vibez/ui';

interface RoomPlayerProps {
  isConnected: boolean;
  castDeviceName: string | null;
  currentSong: Song | null;
  room: Room | null;
  songs: Song[];
  isPlaying: boolean;
  onPlay: () => void;
  onPause: () => void;
  onSkip: () => void;
  onAddSong: () => void;
  onOpenCast: () => void;
  showSpotifyConnect: boolean;
  onConnectSpotify: () => void;
  displayIsPlaying: boolean;
}

export function RoomPlayer({
  isConnected,
  castDeviceName,
  currentSong,
  room,
  songs,
  onPlay,
  onPause,
  onSkip,
  onAddSong,
  onOpenCast,
  showSpotifyConnect,
  onConnectSpotify,
  displayIsPlaying,
}: RoomPlayerProps) {
  return (
    <div className="space-y-6 lg:flex lg:h-full lg:flex-col">
      {/* Player - Reserve height to prevent CLS */}
      <div className="crt-frame relative flex aspect-video min-h-[280px] w-full overflow-hidden rounded-[28px] bg-black sm:min-h-[340px] lg:aspect-auto lg:min-h-0 lg:flex-1">
        {isConnected && castDeviceName && (
          <div className="pointer-events-none absolute inset-0 z-30 flex items-center justify-center">
            <div className="panel-surface flex items-center gap-3 rounded-full px-5 py-2 text-sm text-theme shadow-[0_0_22px_rgba(0,0,0,0.28)]">
              <span className="h-2.5 w-2.5 animate-pulse rounded-full bg-primary" />
              <span className="font-medium">Casting to {castDeviceName}</span>
            </div>
          </div>
        )}
        {currentSong ? (
          currentSong.sourceType === 'spotify' ? (
            <SpotifyPlayer
              onEnded={room?.mode === 'host' ? onSkip : undefined}
              isVisible={!isConnected}
            />
          ) : currentSong.sourceType === 'soundcloud' ? (
            <SoundCloudPlayer
              onEnded={room?.mode === 'host' ? onSkip : undefined}
              isVisible={!isConnected}
            />
          ) : (
            <VideoPlayer
              onEnded={room?.mode === 'host' ? onSkip : undefined}
              isVisible={!isConnected}
            />
          )
        ) : songs.length > 0 ? (
          <div className="absolute inset-0 flex items-center justify-center bg-black">
            {/* SIGNAL CRT */}
            <div className="pointer-events-none absolute inset-0 z-[1] overflow-hidden">
              <div className="vhs-scanlines h-full w-full opacity-[0.2] mix-blend-overlay" />
              <div className="crt-overlay !absolute !z-[2] pointer-events-none inset-0 opacity-[0.1]" />
            </div>
            <div className="relative z-10 text-center">
              <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full border border-theme bg-theme-surface">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-white/30 border-t-white" />
              </div>
              <p className="text-sm text-theme-muted">Loading song...</p>
            </div>
          </div>
        ) : (
          <div className="absolute inset-0 flex items-center justify-center overflow-hidden bg-black">
            {/* SIGNAL CRT */}
            <div className="pointer-events-none absolute inset-0 z-[1] overflow-hidden">
              <div className="vhs-scanlines h-full w-full opacity-[0.2] mix-blend-overlay" />
              <div className="crt-overlay !absolute !z-[2] pointer-events-none inset-0 opacity-[0.1]" />
            </div>
            <div className="relative z-10 text-center">
              <div className="mb-6 inline-flex items-center rounded-full border border-theme px-4 py-2 text-[10px] text-theme-muted tracking-[0.3em]">
                NO SIGNAL
              </div>
              <h3 className="mb-2 font-display text-base text-theme">
                Add a song to light up the room
              </h3>
              <p className="text-theme-muted text-xs">
                Tap "Add Song" to start the music flow.
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Controls (always below video) */}
      <PlayerControls
        isPlaying={displayIsPlaying}
        canPlay={Boolean(currentSong || songs.length > 0)}
        canSkip={Boolean(currentSong)}
        onPlay={onPlay}
        onPause={onPause}
        onSkip={onSkip}
        onAddSong={onAddSong}
        onOpenCast={onOpenCast}
        isCasting={isConnected}
        castDeviceName={castDeviceName}
        showSpotifyConnect={showSpotifyConnect}
        onConnectSpotify={onConnectSpotify}
      />
    </div>
  );
}
