import {
  safeWrapAsync,
  usePlaybackStore,
  useProviderToken,
} from '@vibez/shared';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import SpotifyWebPlayer, {
  type CallbackState,
  type SpotifyPlayer as SpotifySdkPlayer,
} from 'react-spotify-web-playback';
import { AuthOverlay } from './AuthOverlay';

interface Props {
  isVisible?: boolean;
  onEnded?: () => void;
  fill?: boolean;
}

const SpotifyPlayerComponent: React.FC<Props> = ({
  isVisible = true,
  onEnded,
  fill = false,
}) => {
  const currentSong = usePlaybackStore((state) => state.currentSong);
  const isPlaying = usePlaybackStore((state) => state.isPlaying);

  const {
    token: accessToken,
    error: tokenError,
    fetchToken,
  } = useProviderToken();
  const [error, setError] = useState<string | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [isFetchingToken, setIsFetchingToken] = useState(false);
  const lastPositionRef = useRef<number>(0);
  const hasEndedRef = useRef<boolean>(false);
  const sdkPlayerRef = useRef<SpotifySdkPlayer | null>(null);
  const pendingSeekMsRef = useRef<number | null>(null);

  useEffect(() => {
    let isActive = true;

    if (currentSong?.sourceType === 'spotify') {
      setIsFetchingToken(true);
      void fetchToken('spotify').finally(() => {
        if (isActive) setIsFetchingToken(false);
      });
    } else {
      setIsFetchingToken(false);
    }

    return () => {
      isActive = false;
    };
  }, [currentSong?.sourceType, fetchToken]);

  useEffect(() => {
    setIsReady(false);
    hasEndedRef.current = false;
    lastPositionRef.current = 0;
    pendingSeekMsRef.current = usePlaybackStore.getState().actualPositionMs;
    setError(null);
  }, [currentSong?.id]);

  useEffect(() => {
    if (
      !isReady ||
      !sdkPlayerRef.current ||
      pendingSeekMsRef.current === null ||
      !currentSong ||
      currentSong.sourceType !== 'spotify'
    ) {
      return;
    }

    const targetMs = pendingSeekMsRef.current;
    pendingSeekMsRef.current = null;

    if (Math.abs(lastPositionRef.current - targetMs) <= 1000) {
      return;
    }

    void (async () => {
      const player = sdkPlayerRef.current;
      if (!player) return;

      const [seekError] = await safeWrapAsync(player.seek(targetMs));
      if (seekError) {
        console.error('[SpotifyPlayer] Failed to seek:', seekError);
      } else {
        lastPositionRef.current = targetMs;
      }
    })();
  }, [currentSong, isReady]);

  const handleCallback = useCallback(
    (state: CallbackState) => {
      if (state.isActive) {
        setIsReady(true);
      }

      if (
        state.progressMs !== undefined &&
        state.track?.durationMs !== undefined
      ) {
        const isNearEnd = state.progressMs >= state.track.durationMs - 500;
        const wasPlaying = lastPositionRef.current > 0;

        if (
          isNearEnd &&
          wasPlaying &&
          !state.isPlaying &&
          !hasEndedRef.current
        ) {
          hasEndedRef.current = true;
          onEnded?.();
        }

        lastPositionRef.current = state.progressMs;
      }

      if (state.errorType) {
        console.error('[SpotifyPlayer] Error:', state.errorType);
        const errType = String(state.errorType);

        if (
          errType === 'account_error' ||
          errType === 'authentication_error' ||
          errType === 'account'
        ) {
          setError("You don't seem to have premium");
        } else {
          setError('Playback error');
        }
      }
    },
    [onEnded],
  );

  const handleGetPlayer = useCallback((player: SpotifySdkPlayer) => {
    sdkPlayerRef.current = player;
  }, []);

  const handleAuthorize = () => {
    const width = 600;
    const height = 800;
    const left = window.screen.width / 2 - width / 2;
    const top = window.screen.height / 2 - height / 2;

    const popup = window.open(
      '/api/v1/authorizations/spotify',
      'SpotifyAuth',
      `width=${width},height=${height},left=${left},top=${top}`,
    );

    let timer: ReturnType<typeof setInterval> | null = null;

    const cleanup = () => {
      if (timer) clearInterval(timer);
      window.removeEventListener('message', handleMessage);
      fetchToken('spotify', true);
      setError(null);
    };

    const handleMessage = (event: MessageEvent) => {
      console.log('[SpotifyPlayer] Received message:', event.data);
      if (
        event.data?.type === 'oauth-success' &&
        event.data?.provider === 'spotify'
      ) {
        console.log(
          '[SpotifyPlayer] OAuth success message received, cleaning up',
        );
        cleanup();
        popup?.close();
      }
    };

    window.addEventListener('message', handleMessage);

    timer = setInterval(() => {
      if (popup?.closed) {
        console.log('[SpotifyPlayer] Popup closed detected via polling');
        cleanup();
      }
    }, 500);
  };

  if (!currentSong || !isVisible || currentSong.sourceType !== 'spotify') {
    return null;
  }

  const spotifyUri = `spotify:track:${currentSong.sourceId}`;

  const showOverlay =
    (!accessToken && !isFetchingToken) ||
    !!tokenError ||
    (error && (error.includes('auth') || error.includes('premium')));

  const overlayErrorMessage =
    (tokenError?.includes('premium') ? tokenError : null) ||
    (error?.includes('premium') ? error : null) ||
    (error?.includes('auth') ? error : null);

  const containerClass = fill
    ? 'relative h-full w-full overflow-hidden bg-gradient-to-br from-green-900 to-black'
    : 'relative w-full overflow-hidden rounded-xl bg-gradient-to-br from-green-900 to-black';

  const containerStyle = fill
    ? { height: '100%', width: '100%' }
    : { aspectRatio: '16/9', minHeight: '200px' };

  if (!accessToken && isFetchingToken) {
    return (
      <div
        className={`${containerClass} flex items-center justify-center`}
        style={containerStyle}
      >
        <div className="text-center">
          <div className="mx-auto mb-3 h-8 w-8 animate-spin rounded-full border-2 border-green-500/30 border-t-green-500" />
          <p className="text-sm text-white/70">Connecting to Spotify...</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`${containerClass} ${!isVisible ? 'hidden' : ''}`}
      style={containerStyle}
    >
      {showOverlay && (
        <AuthOverlay
          provider="spotify"
          errorMessage={overlayErrorMessage}
          onAuthorize={handleAuthorize}
        />
      )}

      <div className="absolute inset-0 flex items-center justify-center p-8">
        <div className="flex max-w-full items-center gap-6">
          {currentSong.thumbnailUrl && (
            <img
              src={currentSong.thumbnailUrl}
              alt={currentSong.title}
              className="h-32 w-32 shrink-0 rounded-lg object-cover shadow-2xl"
            />
          )}
          <div className="min-w-0">
            <h3 className="truncate font-bold text-white text-xl">
              {currentSong.title}
            </h3>
            <p className="mt-1 truncate text-sm text-white/70">
              {currentSong.artist || 'Unknown Artist'}
            </p>
            <div className="mt-3 flex items-center gap-2">
              <div
                className={`h-2 w-2 rounded-full ${isPlaying ? 'animate-pulse bg-green-500' : 'bg-white/30'}`}
              />
              <span className="text-white/50 text-xs">
                {isPlaying ? 'Playing on Spotify' : 'Paused'}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="absolute right-0 bottom-0 left-0 h-0 overflow-hidden opacity-0">
        {accessToken && (
          <SpotifyWebPlayer
            token={accessToken}
            uris={[spotifyUri]}
            play={isPlaying}
            callback={handleCallback}
            getPlayer={handleGetPlayer}
            initialVolume={0.5}
            name="Vibes Player"
            styles={{
              bgColor: 'transparent',
              color: '#fff',
              trackNameColor: '#fff',
            }}
          />
        )}
      </div>

      {!isReady && !error && !showOverlay && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/50">
          <div className="text-center">
            <div className="mx-auto mb-3 h-8 w-8 animate-spin rounded-full border-2 border-green-500/30 border-t-green-500" />
            <p className="text-sm text-white/70">Loading track...</p>
          </div>
        </div>
      )}
    </div>
  );
};

export const SpotifyPlayer = memo(
  SpotifyPlayerComponent,
  (prevProps, nextProps) => {
    return (
      prevProps.isVisible === nextProps.isVisible &&
      prevProps.onEnded === nextProps.onEnded
    );
  },
);
