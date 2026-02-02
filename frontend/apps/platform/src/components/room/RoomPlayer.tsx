import { usePlayback, usePlaybackPosition, useQueue } from '@vibez/api';
import { type PlaybackState, type Room } from '@vibez/models';
import { safeWrapAsync, usePlaybackStore } from '@vibez/shared';
import { PlayerControls } from '@vibez/ui';
import React, {
  type ComponentType,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useCasting } from '../../hooks/useCasting';
import { useProviderToken } from '../../hooks/useProviderToken';

interface RoomPlayerProps {
  roomId: string;
  displayRoom: Room | null;
  onAddSong: () => void;
  onOpenCast: () => void;
  initialPlayback?: PlaybackState;
}

interface PlayerProps {
  isVisible?: boolean;
  onEnded?: () => void;
  fill?: boolean;
  onNeedsUserGestureChange?: (needsGesture: boolean) => void;
  appContext?: 'platform' | 'cast';
}

type PlayerComponent = ComponentType<PlayerProps>;

interface PlayerLoadErrors {
  spotify: string | null;
  soundcloud: string | null;
  video: string | null;
}

const AutoSkipHandler = ({
  currentSong,
  isPlaying,
  skip,
  mode,
}: {
  currentSong: any;
  isPlaying: boolean;
  skip: () => void;
  mode: string | undefined;
}) => {
  const actualPositionMs = usePlaybackPosition();
  const autoSkipRef = useRef<string | null>(null);

  useEffect(() => {
    if (!currentSong?.id) {
      autoSkipRef.current = null;
      return;
    }
    if (mode !== 'host') return;
    if (!isPlaying || !currentSong.duration) return;

    const durationMs = currentSong.duration * 1000;
    const shouldAutoSkip = actualPositionMs >= durationMs - 750;

    if (shouldAutoSkip && autoSkipRef.current !== currentSong.id) {
      autoSkipRef.current = currentSong.id;
      skip();
    }
  }, [
    actualPositionMs,
    currentSong?.id,
    currentSong?.duration,
    mode,
    isPlaying,
    skip,
  ]);

  return null;
};

export const RoomPlayer = React.memo(
  ({
    roomId,
    displayRoom,
    onAddSong,
    onOpenCast,
    initialPlayback,
  }: RoomPlayerProps) => {
    /* 1. Hooks */
    const { play, pause, skip, fetchPlayback } = usePlayback(roomId);
    const { songs } = useQueue(roomId);
    const { isConnected, castDeviceName } = useCasting(roomId);
    const { token: spotifyToken, fetchToken: fetchSpotifyToken } =
      useProviderToken();

    // Granular store subscriptions
    const isPlaying = usePlaybackStore((state) => state.isPlaying);
    const currentSongFromStore = usePlaybackStore((state) => state.currentSong);

    /* 2. State & Computed */
    const currentSong =
      currentSongFromStore || initialPlayback?.currentSong || null;

    const hasSpotifySongs = useMemo(
      () => songs.some((s) => s.sourceType === 'spotify'),
      [songs],
    );

    const currentSourceType = currentSong?.sourceType ?? null;
    const needsSpotifyPlayer = currentSourceType === 'spotify';
    const needsSoundCloudPlayer = currentSourceType === 'soundcloud';
    const needsVideoPlayer =
      currentSourceType !== null &&
      currentSourceType !== 'spotify' &&
      currentSourceType !== 'soundcloud';

    const [SpotifyPlayerComponent, setSpotifyPlayerComponent] =
      useState<PlayerComponent | null>(null);
    const [SoundCloudPlayerComponent, setSoundCloudPlayerComponent] =
      useState<PlayerComponent | null>(null);
    const [VideoPlayerComponent, setVideoPlayerComponent] =
      useState<PlayerComponent | null>(null);
    const [isPlaybackBlocked, setIsPlaybackBlocked] = useState(false);
    const [playerLoadErrors, setPlayerLoadErrors] = useState<PlayerLoadErrors>({
      spotify: null,
      soundcloud: null,
      video: null,
    });
    const playbackFetchAttemptedRef = useRef<string | null>(null);
    const debugMountRef = useRef(false);

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
      setPlayerLoadErrors((prev) => ({
        ...prev,
        spotify: currentSong?.sourceType === 'spotify' ? null : prev.spotify,
        soundcloud:
          currentSong?.sourceType === 'soundcloud' ? null : prev.soundcloud,
        video:
          currentSong &&
            currentSong.sourceType !== 'spotify' &&
            currentSong.sourceType !== 'soundcloud'
            ? null
            : prev.video,
      }));
    }, [currentSong?.sourceType]);

    useEffect(() => {
      if (hasSpotifySongs) {
        fetchSpotifyToken('spotify');
      }
    }, [hasSpotifySongs, fetchSpotifyToken]);

    useEffect(() => {
      if (!roomId) return;
      if (currentSong || songs.length === 0) return;

      const attemptKey = `${roomId}:${songs.length}`;
      if (playbackFetchAttemptedRef.current === attemptKey) return;
      playbackFetchAttemptedRef.current = attemptKey;

      void fetchPlayback();
    }, [roomId, songs.length, currentSong, fetchPlayback]);

    /* 5. Render */

    useEffect(() => {
      if (!needsSpotifyPlayer || SpotifyPlayerComponent) return;

      let isMounted = true;
      const loadSpotifyPlayer = async () => {
        const [loadErr, module] = await safeWrapAsync(import('@vibez/ui'));
        const resolvedComponent = module?.SpotifyPlayer;
        if (!isMounted || loadErr || !resolvedComponent) {
          if (loadErr) {
            console.error('[RoomPlayer] Spotify player load failed', loadErr);
            setPlayerLoadErrors((prev) => ({
              ...prev,
              spotify: 'Failed to load Spotify player',
            }));
          } else if (!resolvedComponent) {
            setPlayerLoadErrors((prev) => ({
              ...prev,
              spotify: 'Spotify player unavailable',
            }));
          }
          return;
        }
        setSpotifyPlayerComponent(() => resolvedComponent);
      };

      loadSpotifyPlayer();

      return () => {
        isMounted = false;
      };
    }, [needsSpotifyPlayer, SpotifyPlayerComponent]);

    useEffect(() => {
      if (!needsSoundCloudPlayer || SoundCloudPlayerComponent) return;

      let isMounted = true;
      const loadSoundCloudPlayer = async () => {
        const [loadErr, module] = await safeWrapAsync(import('@vibez/ui'));
        const resolvedComponent = module?.SoundCloudPlayer;
        if (!isMounted || loadErr || !resolvedComponent) {
          if (loadErr) {
            console.error(
              '[RoomPlayer] SoundCloud player load failed',
              loadErr,
            );
            setPlayerLoadErrors((prev) => ({
              ...prev,
              soundcloud: 'Failed to load SoundCloud player',
            }));
          } else if (!resolvedComponent) {
            setPlayerLoadErrors((prev) => ({
              ...prev,
              soundcloud: 'SoundCloud player unavailable',
            }));
          }
          return;
        }
        setSoundCloudPlayerComponent(() => resolvedComponent);
      };

      loadSoundCloudPlayer();

      return () => {
        isMounted = false;
      };
    }, [needsSoundCloudPlayer, SoundCloudPlayerComponent]);

    useEffect(() => {
      if (!needsVideoPlayer || VideoPlayerComponent) return;

      let isMounted = true;
      const loadVideoPlayer = async () => {
        const [loadErr, module] = await safeWrapAsync(import('@vibez/ui'));
        const resolvedComponent = module?.VideoPlayer;
        if (!isMounted || loadErr || !resolvedComponent) {
          if (loadErr) {
            console.error('[RoomPlayer] Video player load failed', loadErr);
            setPlayerLoadErrors((prev) => ({
              ...prev,
              video: 'Failed to load video player',
            }));
          } else if (!resolvedComponent) {
            setPlayerLoadErrors((prev) => ({
              ...prev,
              video: 'Video player unavailable',
            }));
          }
          return;
        }
        setVideoPlayerComponent(() => resolvedComponent);
      };

      loadVideoPlayer();

      return () => {
        isMounted = false;
      };
    }, [needsVideoPlayer, VideoPlayerComponent]);

    useEffect(() => {
      if (debugMountRef.current) return;
      debugMountRef.current = true;
      console.log('[RoomPlayer] mount', { roomId });
      return () => {
        console.log('[RoomPlayer] unmount', { roomId });
      };
    }, [roomId]);

    const isSpotifyTrack = currentSong?.sourceType === 'spotify';
    const isSoundCloudTrack = currentSong?.sourceType === 'soundcloud';
    const isVideoTrack = currentSong
      ? currentSong.sourceType !== 'spotify' &&
      currentSong.sourceType !== 'soundcloud'
      : false;
    const isPlayerMissing =
      (isSpotifyTrack && !SpotifyPlayerComponent) ||
      (isSoundCloudTrack && !SoundCloudPlayerComponent) ||
      (isVideoTrack && !VideoPlayerComponent);
    const currentPlayerError = isSpotifyTrack
      ? playerLoadErrors.spotify
      : isSoundCloudTrack
        ? playerLoadErrors.soundcloud
        : isVideoTrack
          ? playerLoadErrors.video
          : null;

    return (
      <div className="space-y-6 lg:flex lg:h-full lg:flex-col">
        <AutoSkipHandler
          currentSong={currentSong}
          isPlaying={isPlaying}
          skip={skip}
          mode={displayRoom?.mode}
        />
        {/* Player - Reserve height to prevent CLS */}
        <div className="crt-frame relative flex min-h-[315px] w-full overflow-hidden rounded-[28px] bg-black sm:min-h-[340px] lg:aspect-auto lg:min-h-0 lg:min-h-[400px] lg:flex-1">
          {VideoPlayerComponent && (
            <div className="absolute inset-0 flex items-center justify-center bg-black">
              <VideoPlayerComponent
                onEnded={displayRoom?.mode === 'host' ? skip : undefined}
                isVisible={!isConnected && isVideoTrack}
                onNeedsUserGestureChange={setIsPlaybackBlocked}
                appContext="platform"
              />
            </div>
          )}
          {isConnected && castDeviceName && (
            <div className="pointer-events-none absolute inset-0 z-30 flex items-center justify-center">
              <div className="panel-surface flex items-center gap-3 rounded-full px-5 py-2 text-sm text-theme shadow-[0_0_22px_rgba(0,0,0,0.28)]">
                <span className="h-2.5 w-2.5 animate-pulse rounded-full bg-primary" />
                <span className="font-medium">Casting to {castDeviceName}</span>
              </div>
            </div>
          )}
          {currentSong ? (
            isPlayerMissing ? (
              <div className="min-h-[315px]">
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
                    <p className="text-sm text-theme-muted">
                      {currentPlayerError ?? 'Loading player...'}
                    </p>
                  </div>
                </div>
              </div>
            ) : isSpotifyTrack ? (
              SpotifyPlayerComponent ? (
                <SpotifyPlayerComponent
                  onEnded={displayRoom?.mode === 'host' ? skip : undefined}
                  isVisible={!isConnected}
                />
              ) : null
            ) : isSoundCloudTrack ? (
              SoundCloudPlayerComponent ? (
                <SoundCloudPlayerComponent
                  onEnded={displayRoom?.mode === 'host' ? skip : undefined}
                  isVisible={!isConnected}
                />
              ) : null
            ) : null
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
          isPlaying={isPlaying && !isPlaybackBlocked}
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
