import React from 'react';
import {
  CastIcon,
  PauseIcon,
  PlayIcon,
  PlusIcon,
  SkipIcon,
  SpotifyIcon,
} from '../icons';

interface Props {
  isPlaying: boolean;
  canPlay: boolean;
  canSkip: boolean;
  onPlay: () => void;
  onPause: () => void;
  onSkip: () => void;
  onAddSong: () => void;
  onOpenCast: () => void;
  isCasting: boolean;
  castDeviceName?: string | null;
  showSpotifyConnect?: boolean;
  onConnectSpotify?: () => void;
}

const PlayerControlsComponent: React.FC<Props> = ({
  isPlaying,
  canPlay,
  canSkip,
  onPlay,
  onPause,
  onSkip,
  onAddSong,
  onOpenCast,
  isCasting,
  castDeviceName,
  showSpotifyConnect,
  onConnectSpotify,
}) => {
  const btnClass =
    'glass p-4 cursor-pointer rounded-xl hover:shadow-retro active:scale-95 transition-all disabled:opacity-30 disabled:cursor-not-allowed group border-2 border-ink/10 dark:border-primary/20 flex items-center justify-center h-14';

  return (
    <div className="w-full">
      <div className="flex items-center justify-start gap-4">
        <button
          onClick={isPlaying ? onPause : onPlay}
          disabled={!canPlay}
          className="group flex h-14 w-14 cursor-pointer items-center justify-center rounded-xl border-2 border-white/50 bg-primary text-white shadow-retro-pink transition-all hover:shadow-neon-pink hover:shadow-retro active:scale-95 disabled:cursor-not-allowed disabled:opacity-30"
        >
          {isPlaying ? (
            <PauseIcon className="h-6 w-6 fill-current" />
          ) : (
            <PlayIcon className="ml-0.5 h-6 w-6 fill-current" />
          )}
        </button>

        <button
          onClick={onSkip}
          disabled={!canSkip}
          className={`${btnClass} w-14`}
          title="Skip"
        >
          <SkipIcon className="h-6 w-6 text-ink/60 transition-colors group-hover:text-primary dark:text-dark-text-muted dark:group-hover:text-primary" />
        </button>

        <button
          onClick={onOpenCast}
          className={`${btnClass} w-14 ${isCasting ? 'border-primary/20 bg-primary/10 dark:border-primary/30 dark:bg-primary/20' : ''}`}
          title={
            isCasting && castDeviceName
              ? `Casting to ${castDeviceName}`
              : 'Cast'
          }
        >
          <CastIcon
            className={`h-6 w-6 transition-colors ${isCasting ? 'text-primary dark:text-primary-light' : 'text-ink/60 group-hover:text-primary dark:text-dark-text-muted dark:group-hover:text-primary'}`}
            showDot={isCasting}
          />
        </button>

        {showSpotifyConnect && onConnectSpotify && (
          <button
            onClick={onConnectSpotify}
            className={`${btnClass} ml-auto gap-2 px-4 text-[#1DB954] hover:border-[#1DB954]/30 hover:bg-[#1DB954]/10 dark:hover:bg-[#1DB954]/20`}
            title="Connect Spotify"
          >
            <SpotifyIcon className="h-6 w-6" />
            <span className="whitespace-nowrap font-black text-sm tracking-wide">
              Connect Spotify
            </span>
          </button>
        )}

        <button
          onClick={onAddSong}
          className={`${btnClass} ${!showSpotifyConnect ? 'ml-auto' : ''} gap-2 px-6 text-primary hover:border-primary/30 dark:text-primary-light dark:hover:border-primary/40`}
          title="Add Song"
        >
          <PlusIcon className="h-5 w-5 shrink-0" />
          <span className="whitespace-nowrap font-black text-ink text-sm tracking-wide dark:text-dark-text">
            Add Song
          </span>
        </button>
      </div>

      {isCasting && castDeviceName && (
        <div className="mt-3 flex items-center justify-center gap-2 text-primary text-xs dark:text-primary-light">
          <div className="h-2 w-2 animate-pulse rounded-full bg-primary dark:bg-primary-light" />
          <span className="font-medium">Casting to {castDeviceName}</span>
        </div>
      )}
    </div>
  );
};

export const PlayerControls = React.memo(PlayerControlsComponent);
