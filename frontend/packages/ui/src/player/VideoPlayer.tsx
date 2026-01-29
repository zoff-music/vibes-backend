import { useProviderToken } from '@vibez/api';
import { usePlaybackStore } from '@vibez/shared';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import YouTube, { type YouTubeProps } from 'react-youtube';
import { AuthOverlay } from './AuthOverlay';

interface Props {
  isVisible?: boolean;
  onEnded?: () => void;
  fill?: boolean;
}

interface YouTubePlayerRef {
  seekTo: (seconds: number, allowSeekAhead?: boolean) => void;
  getCurrentTime: () => number;
  playVideo: () => void;
  pauseVideo: () => void;
  getPlayerState: () => number;
}

const VideoPlayerComponent = ({
  isVisible = true,
  onEnded,
  fill = false,
}: Props) => {
  const currentSong = usePlaybackStore((state) => state.currentSong);
  const isPlaying = usePlaybackStore((state) => state.isPlaying);

  const { fetchToken } = useProviderToken();
  const playerRef = useRef<YouTubePlayerRef | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [isVerifying, setIsVerifying] = useState(false);

  const handleAuthorize = () => {
    // Check if running on Chromecast (CrKey)
    const isChromecast = /CrKey/i.test(navigator.userAgent);
    if (isChromecast) {
      setError('Please authorize YouTube on your phone to continue casting.');
      return;
    }

    const width = 600;
    const height = 800;
    const left = window.screen.width / 2 - width / 2;
    const top = window.screen.height / 2 - height / 2;

    const popup = window.open(
      '/api/v1/authorizations/youtube',
      'YouTubeAuth',
      `width=${width},height=${height},left=${left},top=${top}`,
    );

    let timer: ReturnType<typeof setInterval> | null = null;

    const cleanup = () => {
      if (timer) clearInterval(timer);
      window.removeEventListener('message', handleMessage);

      setIsVerifying(true);
      setError(null);

      fetchToken('youtube', true).then((newToken: string | null) => {
        setIsVerifying(false);
        if (!newToken) {
          setError('Failed to refresh token after authorization.');
        }
        if (playerRef.current) {
        }
      });
    };

    const handleMessage = (event: MessageEvent) => {
      if (
        event.data?.type === 'oauth-success' &&
        event.data?.provider === 'youtube'
      ) {
        cleanup();
        popup?.close();
      }
    };

    window.addEventListener('message', handleMessage);

    timer = setInterval(() => {
      if (popup?.closed) {
        cleanup();
      }
    }, 1000);
  };

  useEffect(() => {
    setIsReady(false);
    setError(null);
    playerRef.current = null;
  }, [currentSong?.id]);

  useEffect(() => {
    const interval = setInterval(() => {
      if (!isReady || !playerRef.current) {
        return;
      }

      try {
        const player = playerRef.current;
        const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
        const targetTime = actualPositionMs / 1000;

        const currentTime = player.getCurrentTime();
        const drift = Math.abs(currentTime - targetTime);

        if (drift > 2) {
          player.seekTo(targetTime, true);
        }
      } catch (_err) {}
    }, 1000);

    return () => clearInterval(interval);
  }, [isReady, isPlaying]);

  useEffect(() => {
    if (!isReady || !playerRef.current) return;

    try {
      const player = playerRef.current;
      const state = player.getPlayerState();

      if (isPlaying && state !== 1 && state !== 3) {
        const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
        const targetTime = actualPositionMs / 1000;
        const currentTime = player.getCurrentTime();
        if (Math.abs(currentTime - targetTime) > 1) {
          player.seekTo(targetTime, true);
        }
        player.playVideo();
      } else if (!isPlaying && state === 1) {
        player.pauseVideo();
      }
    } catch (_err) {}
  }, [isReady, isPlaying]);

  const handleReady = useCallback((event: { target: YouTubePlayerRef }) => {
    playerRef.current = event.target;
    setIsReady(true);
    setError(null);

    const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
    if (actualPositionMs > 0) {
      const targetTime = actualPositionMs / 1000;
      event.target.seekTo(targetTime, true);
    }
  }, []);

  const handleStateChange = useCallback((event: { data: number }) => {
    const state = event.data;

    if (state === 1 || state === 3) {
      setIsReady(true);
    }
  }, []);

  const handleEnd = useCallback(() => {
    onEnded?.();
  }, [onEnded]);

  const handleError = useCallback(
    async (event: unknown) => {
      console.error('[VideoPlayer] Player error:', event);

      const fetchedToken = await fetchToken('youtube');

      if (!fetchedToken) {
        setError('Authorization required or video unavailable');
      } else {
        setError('Failed to load video even with authorization');
      }
    },
    [fetchToken],
  );

  if (!currentSong || !isVisible) {
    return null;
  }

  const videoId =
    currentSong.sourceType === 'youtube' ? currentSong.sourceId : null;

  if (!videoId) {
    return null;
  }

  const opts: YouTubeProps['opts'] = {
    height: '100%',
    width: '100%',
    playerVars: {
      autoplay: 1,
      controls: 0,
      disablekb: 1,
      fs: 0,
      modestbranding: 1,
      rel: 0,
      playsinline: 1,
    },
  };

  const showOverlay = !!error;

  const containerClass = fill
    ? 'relative h-full w-full overflow-hidden bg-black/40 backdrop-blur-md [&_iframe]:h-full [&_iframe]:w-full [&_iframe]:scale-[1.65] [&_iframe]:origin-center'
    : 'relative w-full overflow-hidden rounded-xl bg-black/40 backdrop-blur-md';

  const containerStyle = fill
    ? { height: '100%', width: '100%' }
    : { aspectRatio: '16/9', minHeight: '200px' };

  // Main render logic always includes the container and CRT layers
  return (
    <div
      className={`${containerClass} ${!isVisible ? 'hidden' : ''}`}
      style={containerStyle}
    >
      {/* Video Content - Back Layer */}
      {!isVerifying && videoId && (
        <YouTube
          videoId={videoId}
          opts={opts}
          onReady={handleReady}
          onStateChange={handleStateChange}
          onEnd={handleEnd}
          onError={handleError}
          className="absolute inset-0 h-full w-full"
        />
      )}

      {/* CRT Effects Layer - Middle Layer (if shown) */}
      {showOverlay && (
        <div className="pointer-events-none absolute inset-0 z-[5] overflow-hidden">
          <div className="vhs-scanlines h-full w-full opacity-[0.14] mix-blend-overlay" />
          <div className="crt-overlay !absolute !z-[6] pointer-events-none inset-0 opacity-[0.1]" />
        </div>
      )}

      {/* Auth/Error Overlay - Top Layer */}
      {showOverlay && (
        <AuthOverlay
          provider="youtube"
          errorMessage={error}
          onAuthorize={handleAuthorize}
        />
      )}

      {/* Loading States */}
      {isVerifying && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="text-center">
            <div className="mx-auto mb-3 h-8 w-8 animate-spin rounded-full border-2 border-red-500/30 border-t-red-500" />
            <p className="font-mono text-[10px] text-white/70 uppercase tracking-widest">
              Verifying Authorization...
            </p>
          </div>
        </div>
      )}

      {!isReady && !showOverlay && !isVerifying && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="text-center">
            <div className="mx-auto mb-3 h-8 w-8 animate-spin rounded-full border-2 border-red-500/30 border-t-red-500" />
            <p className="font-mono text-[10px] text-white/70 uppercase tracking-widest">
              Loading Satellite Feed...
            </p>
          </div>
        </div>
      )}
    </div>
  );
};

export const VideoPlayer = memo(VideoPlayerComponent);
