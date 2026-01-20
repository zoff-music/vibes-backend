import { memo, useCallback, useEffect, useRef, useState } from 'react';
import SpotifyWebPlayer, {
  type CallbackState,
} from 'react-spotify-web-playback';
import { useProviderToken, usePlaybackStore } from '@vibez/shared';
import { AuthOverlay } from './AuthOverlay';

interface Props {
  isVisible?: boolean;
  onEnded?: () => void;
}

const SpotifyPlayerComponent: React.FC<Props> = ({
  isVisible = true,
  onEnded,
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
  const lastPositionRef = useRef<number>(0);
  const hasEndedRef = useRef<boolean>(false);

  // Fetch access token on mount or when song changes
  useEffect(() => {
    if (currentSong?.sourceType === 'spotify') {
      fetchToken('spotify');
    }
  }, [currentSong?.sourceType, fetchToken]);

  // Reset state when song changes
  useEffect(() => {
    setIsReady(false);
    hasEndedRef.current = false;
    lastPositionRef.current = 0;
    setError(null);
  }, [currentSong?.id]);

  // Handle player state changes
  const handleCallback = useCallback(
    (state: CallbackState) => {
      // Track ready state
      if (state.isActive) {
        setIsReady(true);
      }

      // Check if track ended
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

      // Handle errors
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

  const handleAuthorize = () => {
    // Open popup for auth
    const width = 600;
    const height = 800;
    const left = window.screen.width / 2 - width / 2;
    const top = window.screen.height / 2 - height / 2;

    // Using direct URL for improved reliability with popups
    const popup = window.open(
      '/api/v1/authorizations/spotify',
      'SpotifyAuth',
      `width=${width},height=${height},left=${left},top=${top}`,
    );

    let timer: ReturnType<typeof setInterval> | null = null;

    // Cleanup function
    const cleanup = () => {
      if (timer) clearInterval(timer);
      window.removeEventListener('message', handleMessage);
      // Re-fetch token and clear error
      fetchToken('spotify', true);
      setError(null);
    };

    // Message handler
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

    // Polling fallback
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

  // Show auth overlay if we have an error related to auth, premium, OR completely missing token
  const showOverlay =
    error &&
    (error.includes('auth') || error.includes('premium') || !accessToken);

  if (showOverlay) {
    return (
      <div
        className="relative w-full overflow-hidden rounded-xl bg-black"
        style={{ aspectRatio: '16/9' }}
      >
        <AuthOverlay
          provider="spotify"
          errorMessage={
            (tokenError?.includes('premium') ? tokenError : null) ||
            (error?.includes('premium') ? error : null)
          }
          onAuthorize={handleAuthorize}
        />
      </div>
    );
  }

  if (tokenError) {
    return (
      <div className="relative flex aspect-video w-full items-center justify-center overflow-hidden rounded-xl bg-black">
        <AuthOverlay
          provider="spotify"
          errorMessage={tokenError.includes('premium') ? tokenError : null}
          onAuthorize={handleAuthorize}
        />
      </div>
    );
  }

  if (!accessToken) {
    return (
      <div className="relative flex aspect-video w-full items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-green-900 to-black">
        <div className="text-center">
          <div className="mx-auto mb-3 h-8 w-8 animate-spin rounded-full border-2 border-green-500/30 border-t-green-500" />
          <p className="text-sm text-white/70">Connecting to Spotify...</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`relative w-full overflow-hidden rounded-xl bg-gradient-to-br from-green-900 to-black ${!isVisible ? 'hidden' : ''}`}
      style={{
        aspectRatio: '16/9',
        minHeight: '200px',
      }}
    >
      {/* Album art and track info overlay */}
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

      {/* Hidden Spotify player - controls playback */}
      <div className="absolute right-0 bottom-0 left-0 h-0 overflow-hidden opacity-0">
        <SpotifyWebPlayer
          token={accessToken}
          uris={[spotifyUri]}
          play={isPlaying}
          callback={handleCallback}
          initialVolume={0.5}
          name="Vibes Player"
          styles={{
            bgColor: 'transparent',
            color: '#fff',
            trackNameColor: '#fff',
          }}
        />
      </div>

      {/* Loading overlay */}
      {!isReady && !error && (
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
