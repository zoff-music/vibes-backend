import { usePlaybackStore } from '@vibez/shared';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import ReactPlayer from 'react-player';

interface Props {
  isVisible?: boolean;
  onEnded?: () => void;
  fill?: boolean;
}

const SoundCloudPlayerComponent: React.FC<Props> = ({
  isVisible = true,
  onEnded,
  fill = false,
}) => {
  const currentSong = usePlaybackStore((state) => state.currentSong);
  const isPlaying = usePlaybackStore((state) => state.isPlaying);

  const playerRef = useRef<any>(null);
  const [isReady, setIsReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setIsReady(false);
    setError(null);
  }, [currentSong?.id]);

  useEffect(() => {
    const interval = setInterval(() => {
      if (!isReady || !playerRef.current) {
        return;
      }

      try {
        const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
        const targetTime = actualPositionMs / 1000;
        const currentTime = playerRef.current.getCurrentTime();
        const drift = Math.abs(currentTime - targetTime);

        if (drift > 2) {
          console.log('[SoundCloudPlayer] Seeking due to drift:', {
            currentTime,
            targetTime,
            drift,
          });
          playerRef.current.seekTo(targetTime, 'seconds');
        }
      } catch (err) {
        console.warn('[SoundCloudPlayer] Error syncing position:', err);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isReady, isPlaying]);

  const handleReady = useCallback(() => {
    setIsReady(true);
    setError(null);

    const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
    if (actualPositionMs > 0 && playerRef.current) {
      const targetTime = actualPositionMs / 1000;
      playerRef.current.seekTo(targetTime, 'seconds');
    }
  }, []);

  const handleEnded = useCallback(() => {
    onEnded?.();
  }, [onEnded]);

  if (!currentSong || !isVisible) {
    return null;
  }

  if (currentSong.sourceType !== 'soundcloud') {
    return null;
  }

  const soundcloudUrl = `https://api.soundcloud.com/tracks/${currentSong.sourceId}`;

  const containerClass = fill
    ? 'relative h-full w-full overflow-hidden bg-gradient-to-br from-orange-900 to-black'
    : 'relative w-full overflow-hidden rounded-xl bg-gradient-to-br from-orange-900 to-black';

  const containerStyle = fill
    ? { height: '100%', width: '100%' }
    : { aspectRatio: '16/9', minHeight: '200px' };

  return (
    <div
      className={`${containerClass} ${!isVisible ? 'hidden' : ''}`}
      style={containerStyle}
    >
      {error && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-black/90">
          <div className="px-4 text-center">
            <p className="mb-2 text-error text-sm">SoundCloud Error</p>
            <p className="text-text-muted text-xs">{error}</p>
          </div>
        </div>
      )}

      <div className="absolute inset-0 z-10 flex items-center justify-center p-8">
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
                className={`h-2 w-2 rounded-full ${isPlaying ? 'animate-pulse bg-orange-500' : 'bg-white/30'}`}
              />
              <span className="text-white/50 text-xs">
                {isPlaying ? 'Playing on SoundCloud' : 'Paused'}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="absolute right-0 bottom-0 left-0 h-0 overflow-hidden opacity-0">
        <ReactPlayer
          key={currentSong.id}
          ref={playerRef}
          src={soundcloudUrl}
          playing={isPlaying}
          controls={false}
          width="100%"
          height="100%"
          onReady={handleReady}
          onEnded={handleEnded}
        />
      </div>

      {!isReady && !error && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-black/50">
          <div className="text-center">
            <div className="mx-auto mb-3 h-8 w-8 animate-spin rounded-full border-2 border-orange-500/30 border-t-orange-500" />
            <p className="text-sm text-white/70">Loading track...</p>
          </div>
        </div>
      )}
    </div>
  );
};

export const SoundCloudPlayer = memo(
  SoundCloudPlayerComponent,
  (prevProps, nextProps) => {
    return (
      prevProps.isVisible === nextProps.isVisible &&
      prevProps.onEnded === nextProps.onEnded
    );
  },
);
