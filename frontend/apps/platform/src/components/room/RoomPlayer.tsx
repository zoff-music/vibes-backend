import { usePlayback, useQueue } from '@vibez/api';
import { type PlaybackState, type Room } from '@vibez/models';
import { usePlaybackStore } from '@vibez/shared';
import {
  PlayerControls,
  SoundCloudPlayer,
  SpotifyPlayer,
  VideoPlayer,
} from '@vibez/ui';
import React, { useCallback, useEffect, useMemo } from 'react';
import { useCasting } from '../../hooks/useCasting';
import { useProviderToken } from '../../hooks/useProviderToken';

interface RoomPlayerProps {
  roomId: string;
  displayRoom: Room | null;
  onAddSong: () => void;
  onOpenCast: () => void;
  initialPlayback?: PlaybackState;
}

export const RoomPlayer = React.memo(
  ({
    roomId,
    displayRoom,
    onAddSong,
    onOpenCast,
    initialPlayback,
  }: RoomPlayerProps) => {
    /* 1. Hooks */
    const { play, pause, skip } = usePlayback(roomId);
    const { songs } = useQueue(roomId);
    const { isConnected, castDeviceName } = useCasting(roomId);
    const { token: spotifyToken, fetchToken: fetchSpotifyToken } =
      useProviderToken();

    // Granular store subscriptions
    const currentSongFromStore = usePlaybackStore((state) => state.currentSong);
    const isPlayingFromStore = usePlaybackStore((state) => state.isPlaying);

    /* 2. State & Computed */
    const currentSong =
      currentSongFromStore || initialPlayback?.currentSong || null;
    const isPlaying =
      isPlayingFromStore !== undefined
        ? isPlayingFromStore
        : initialPlayback?.isPlaying || false;

    const hasSpotifySongs = useMemo(
      () => songs.some((s) => s.sourceType === 'spotify'),
      [songs],
    );

    /* 3. Handlers */
    const handleConnectSpotify = useCallback(() => {
      const width = 600;
      const height = 800;
      const left = window.screen.width / 2 - width / 2;
      const top = window.screen.height / 2 - height / 2;

      const popup = window.open(
        '/api/v1/authorizations/spotify',
        'SpotifyAuth',
        `width=${width},height=${height},left=${left},top=${top}`,
      );

      const handleMessage = (event: MessageEvent) => {
        if (
          event.data?.type === 'oauth-success' &&
          event.data?.provider === 'spotify'
        ) {
          fetchSpotifyToken('spotify', true);
          popup?.close();
          window.removeEventListener('message', handleMessage);
        }
      };

      window.addEventListener('message', handleMessage);

      const timer = setInterval(() => {
        if (popup?.closed) {
          window.removeEventListener('message', handleMessage);
          clearInterval(timer);
          fetchSpotifyToken('spotify', true);
        }
      }, 1000);
    }, [fetchSpotifyToken]);

    /* 4. Effects */
    useEffect(() => {
      if (hasSpotifySongs) {
        fetchSpotifyToken('spotify');
      }
    }, [hasSpotifySongs, fetchSpotifyToken]);
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
                onEnded={displayRoom?.mode === 'host' ? skip : undefined}
                isVisible={!isConnected}
              />
            ) : currentSong.sourceType === 'soundcloud' ? (
              <SoundCloudPlayer
                onEnded={displayRoom?.mode === 'host' ? skip : undefined}
                isVisible={!isConnected}
              />
            ) : (
              <VideoPlayer
                onEnded={displayRoom?.mode === 'host' ? skip : undefined}
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
          isPlaying={isPlaying}
          canPlay={Boolean(currentSong || songs.length > 0)}
          canSkip={Boolean(currentSong)}
          onPlay={play}
          onPause={pause}
          onSkip={skip}
          onAddSong={onAddSong}
          onOpenCast={onOpenCast}
          isCasting={isConnected}
          castDeviceName={castDeviceName}
          showSpotifyConnect={hasSpotifySongs && !spotifyToken}
          onConnectSpotify={handleConnectSpotify}
        />
      </div>
    );
  },
);
