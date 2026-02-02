import { useProviderToken } from '@vibez/api';
import { usePlaybackStore } from '@vibez/shared';
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
  loadVideoById: (videoId: string, startSeconds?: number) => void;
  mute: () => void;
  unMute: () => void;
  isMuted: () => boolean;
}

const MAX_AUTOPLAY_RETRIES = 12;
const AUTOPLAY_RETRY_MS = 500;
const AUTOPLAY_KICK_COOLDOWN_MS = 800;
const DEBUG = true;

const VideoPlayerComponent = ({
  isVisible = true,
  onEnded,
  fill = false,
}: Props) => {
  const currentSong = usePlaybackStore((state) => state.currentSong);
  const isPlaying = usePlaybackStore((state) => state.isPlaying);

  const { fetchToken } = useProviderToken();
  const playerRef = useRef<YouTubePlayerRef | null>(null);
  const lastVideoIdRef = useRef<string | null>(null);
  const lastLoadedVideoIdRef = useRef<string | null>(null);
  const initialVideoIdRef = useRef<string | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [isVerifying, setIsVerifying] = useState(false);
  const autoPlayRetryRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const autoPlayKickCountRef = useRef(0);
  const autoPlayKickLastAtRef = useRef(0);
  const autoPlayKickVideoIdRef = useRef<string | null>(null);
  const isYouTubeActive = currentSong?.sourceType === 'youtube';
  const shouldPlay = isYouTubeActive && isPlaying;
  const videoId =
    currentSong?.sourceType === 'youtube' ? currentSong.sourceId : null;
  const debugLastRef = useRef(0);
  const hasEverPlayedRef = useRef(false);
  const [needsUserGesture, setNeedsUserGesture] = useState(false);
  const [isMutedState, setIsMutedState] = useState(false);
  const origin =
    typeof window === 'undefined' ? undefined : window.location.origin;

  const debugLog = useCallback(
    (label: string, extra?: Record<string, unknown>) => {
      if (!DEBUG) return;
      const now = Date.now();
      const isUnmuteLog = label.startsWith('unmute-');
      if (!isUnmuteLog && now - debugLastRef.current < 250) return;
      if (!isUnmuteLog) {
        debugLastRef.current = now;
      }
      const visibility =
        typeof document === 'undefined' ? 'unknown' : document.visibilityState;
      const playerState = playerRef.current?.getPlayerState?.();
      const muted = playerRef.current?.isMuted?.();
      const payload = {
        videoId,
        resolvedVideoId: videoId ?? lastVideoIdRef.current,
        isPlaying,
        shouldPlay,
        isReady,
        visibility,
        playerState,
        muted,
        ...extra,
      };
      console.log('[VideoPlayer]', label, JSON.stringify(payload));
    },
    [videoId, isPlaying, shouldPlay, isReady],
  );

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
    setError(null);
    if (!playerRef.current) {
      setIsReady(false);
    }
    if (isYouTubeActive && !hasEverPlayedRef.current) {
      setNeedsUserGesture(true);
    }
    debugLog('song-change', { currentSongId: currentSong?.id });
  }, [currentSong?.id, isYouTubeActive]);

  useEffect(() => {
    debugLog('mount');
    return () => {
      debugLog('unmount');
    };
  }, []);

  useEffect(() => {
    if (!currentSong && !isPlaying) {
      lastVideoIdRef.current = null;
      lastLoadedVideoIdRef.current = null;
    }
  }, [currentSong, isPlaying]);

  useEffect(() => {
    if (!isReady || !playerRef.current || !shouldPlay) return;

    const interval = setInterval(() => {
      if (!isReady || !playerRef.current) {
        return;
      }
      if (typeof document !== 'undefined' && document.hidden) {
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
  }, [isReady, shouldPlay]);

  useEffect(() => {
    if (!isReady || !playerRef.current) return;

    const syncPlaybackState = () => {
      try {
        const player = playerRef.current;
        if (!player) return;

        const state = player.getPlayerState();

        if (shouldPlay) {
          const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
          const targetTime = actualPositionMs / 1000;
          const currentTime = player.getCurrentTime();
          if (Math.abs(currentTime - targetTime) > 1) {
            player.seekTo(targetTime, true);
          }
          if (state !== 1 && state !== 3) {
            player.playVideo();
          }
        } else if (state === 1) {
          player.pauseVideo();
        }
      } catch (_err) {}
    };

    syncPlaybackState();

    const handleVisibilityChange = () => {
      if (typeof document === 'undefined') return;
      debugLog('visibilitychange', { visibility: document.visibilityState });
      if (document.visibilityState !== 'visible') return;
      syncPlaybackState();
    };

    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', handleVisibilityChange);
    }

    return () => {
      if (typeof document !== 'undefined') {
        document.removeEventListener(
          'visibilitychange',
          handleVisibilityChange,
        );
      }
    };
  }, [isReady, shouldPlay]);

  useEffect(() => {
    if (isYouTubeActive || !playerRef.current) return;
    try {
      playerRef.current.pauseVideo();
    } catch (_err) {}
  }, [isYouTubeActive]);

  const kickAutoplay = useCallback(
    (reason: 'state' | 'hidden' | 'retry') => {
      if (!videoId || !shouldPlay) return;
      const player = playerRef.current;
      if (!player) return;

      if (autoPlayKickVideoIdRef.current !== videoId) {
        autoPlayKickVideoIdRef.current = videoId;
        autoPlayKickCountRef.current = 0;
      }

      const now = Date.now();
      if (now - autoPlayKickLastAtRef.current < AUTOPLAY_KICK_COOLDOWN_MS) {
        return;
      }
      if (autoPlayKickCountRef.current >= MAX_AUTOPLAY_RETRIES) {
        return;
      }

      autoPlayKickLastAtRef.current = now;
      autoPlayKickCountRef.current += 1;

      try {
        const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
        const startSeconds = actualPositionMs > 0 ? actualPositionMs / 1000 : 0;
        if (reason !== 'state' || player.getPlayerState() !== 1) {
          player.loadVideoById(videoId, startSeconds);
        }
        player.playVideo();
        debugLog('kick', { reason, attempts: autoPlayKickCountRef.current });
      } catch (_err) {}
    },
    [videoId, shouldPlay],
  );

  useEffect(() => {
    if (!videoId || !shouldPlay) {
      if (autoPlayRetryRef.current) {
        clearInterval(autoPlayRetryRef.current);
        autoPlayRetryRef.current = null;
      }
      autoPlayKickCountRef.current = 0;
      autoPlayKickVideoIdRef.current = null;
      return;
    }

    let attempts = 0;
    const attemptPlay = () => {
      const player = playerRef.current;
      if (!player) return;
      try {
        const state = player.getPlayerState();
        if (state === 1 || state === 3) {
          return;
        }
        kickAutoplay('retry');
      } catch (_err) {}
    };

    attemptPlay();

    autoPlayRetryRef.current = setInterval(() => {
      attempts += 1;
      attemptPlay();
      if (attempts >= MAX_AUTOPLAY_RETRIES) {
        if (autoPlayRetryRef.current) {
          clearInterval(autoPlayRetryRef.current);
          autoPlayRetryRef.current = null;
        }
      }
    }, AUTOPLAY_RETRY_MS);

    return () => {
      if (autoPlayRetryRef.current) {
        clearInterval(autoPlayRetryRef.current);
        autoPlayRetryRef.current = null;
      }
    };
  }, [videoId, shouldPlay, kickAutoplay]);

  useEffect(() => {
    if (!videoId || !shouldPlay) return;
    if (typeof document === 'undefined') return;
    if (document.visibilityState !== 'hidden') return;

    let attempts = 0;
    const kickInterval = setInterval(() => {
      attempts += 1;
      try {
        const state = playerRef.current?.getPlayerState();
        if (state === 1 || state === 3) {
          clearInterval(kickInterval);
          return;
        }
        kickAutoplay('hidden');
      } catch (_err) {}

      if (attempts >= MAX_AUTOPLAY_RETRIES) {
        clearInterval(kickInterval);
      }
    }, AUTOPLAY_RETRY_MS);

    return () => clearInterval(kickInterval);
  }, [videoId, shouldPlay, kickAutoplay]);

  const handleReady = useCallback((event: { target: YouTubePlayerRef }) => {
    playerRef.current = event.target;
    setIsReady(true);
    setError(null);
    debugLog('ready');

    const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
    if (actualPositionMs > 0) {
      const targetTime = actualPositionMs / 1000;
      event.target.seekTo(targetTime, true);
    }
    const activeSong = usePlaybackStore.getState().currentSong;
    if (activeSong?.sourceType === 'youtube') {
      lastLoadedVideoIdRef.current = activeSong.sourceId;
    }

    if (usePlaybackStore.getState().isPlaying) {
      try {
        if (!hasEverPlayedRef.current) {
          event.target.mute();
          setIsMutedState(true);
        }
        event.target.playVideo();
      } catch (_err) {}
    }
  }, []);

  const handleStateChange = useCallback(
    (event: { data: number }) => {
      const state = event.data;
      debugLog('state', { state });

      if (state === 1 || state === 3) {
        setIsReady(true);
        const muted = playerRef.current?.isMuted?.() ?? false;
        setIsMutedState(muted);
        if (!muted) {
          hasEverPlayedRef.current = true;
          setNeedsUserGesture(false);
        } else {
          setNeedsUserGesture(true);
        }
        return;
      }

      if (state === 5 || state === -1) {
        kickAutoplay('state');
      } else if (state === 2 && shouldPlay) {
        try {
          playerRef.current?.playVideo();
        } catch (_err) {}
      }
    },
    [kickAutoplay, shouldPlay],
  );

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

  if (videoId) {
    lastVideoIdRef.current = videoId;
  }

  const resolvedVideoId = videoId ?? lastVideoIdRef.current;
  if (!initialVideoIdRef.current && resolvedVideoId) {
    initialVideoIdRef.current = resolvedVideoId;
  }
  const youtubeVideoIdProp = initialVideoIdRef.current ?? resolvedVideoId;

  useEffect(() => {
    if (!videoId) return;
    const player = playerRef.current;
    if (!player) return;
    if (lastLoadedVideoIdRef.current === videoId) return;

    const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
    const startSeconds = actualPositionMs > 0 ? actualPositionMs / 1000 : 0;
    try {
      player.loadVideoById(videoId, startSeconds);
      lastLoadedVideoIdRef.current = videoId;
      debugLog('load-video', { startSeconds });
    } catch (_err) {}
  }, [videoId, isReady]);

  if (!resolvedVideoId) {
    return null;
  }

  const opts: YouTubeProps['opts'] = useMemo(
    () => ({
      height: '100%',
      width: '100%',
      playerVars: {
        autoplay: 1,
        controls: 0,
        disablekb: 1,
        enablejsapi: 1,
        fs: 0,
        modestbranding: 1,
        rel: 0,
        playsinline: 1,
        origin,
      },
    }),
    [origin],
  );

  const showOverlay = !!error;
  const showClickToPlay =
    isYouTubeActive &&
    (needsUserGesture || (shouldPlay && isMutedState)) &&
    !error &&
    !isVerifying;

  const containerClass = fill
    ? 'relative h-full w-full overflow-hidden bg-black/40 backdrop-blur-md [&_iframe]:h-full [&_iframe]:w-full [&_iframe]:scale-[1.65] [&_iframe]:origin-center'
    : 'relative w-full overflow-hidden rounded-xl bg-black/40 backdrop-blur-md';

  const containerStyle = fill
    ? { height: '100%', width: '100%' }
    : { aspectRatio: '16/9', minHeight: '200px' };

  // Main render logic always includes the container and CRT layers
  return (
    <div
      className={`${containerClass} ${
        !isVisible ? 'pointer-events-none opacity-0' : ''
      }`}
      style={containerStyle}
    >
      {/* Video Content - Back Layer */}
      {!isVerifying && youtubeVideoIdProp && (
        <YouTube
          videoId={youtubeVideoIdProp}
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

      {showClickToPlay && (
        <button
          type="button"
          className="absolute inset-0 z-20 flex items-center justify-center bg-black/70 backdrop-blur-sm"
          onClick={() => {
            const player = playerRef.current;
            if (!player) return;
            try {
              player.unMute();
              setIsMutedState(false);
              player.playVideo();
              hasEverPlayedRef.current = true;
              setNeedsUserGesture(false);
              debugLog('user-gesture-play');
            } catch (_err) {}
          }}
        >
          <div className="text-center">
            <div className="mx-auto mb-3 inline-flex h-12 w-12 items-center justify-center rounded-full border border-white/30 text-white/80">
              ▶
            </div>
            <p className="font-mono text-[11px] text-white/80 uppercase tracking-widest">
              Click to play
            </p>
          </div>
        </button>
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
