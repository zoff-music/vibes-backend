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
    'panel-surface cursor-pointer rounded-2xl hover:shadow-[0_0_18px_rgba(255,46,151,0.25)] active:scale-95 transition-all disabled:opacity-40 disabled:cursor-not-allowed group border border-theme flex items-center justify-center h-12';

  return (
    <div className="w-full">
      <div className="flex items-center justify-start gap-4">
        <button
          onClick={isPlaying ? onPause : onPlay}
          disabled={!canPlay}
          className="group flex h-12 w-12 cursor-pointer items-center justify-center rounded-2xl border border-primary/60 bg-primary text-white shadow-[0_0_24px_rgba(255,46,151,0.45)] transition-all hover:shadow-[0_0_30px_rgba(255,46,151,0.6)] active:scale-95 disabled:cursor-not-allowed disabled:opacity-40"
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
          className={`${btnClass} w-12`}
          title="Skip"
        >
          <SkipIcon className="h-5 w-5 text-theme-muted transition-colors group-hover:text-primary" />
        </button>

        <button
          onClick={onOpenCast}
          className={`${btnClass} w-12 ${isCasting ? 'border-primary/40 bg-primary/10' : ''}`}
          title={
            isCasting && castDeviceName
              ? `Casting to ${castDeviceName}`
              : 'Cast'
          }
        >
          <CastIcon
            className={`h-5 w-5 transition-colors ${isCasting ? 'text-primary' : 'text-theme-muted group-hover:text-primary'}`}
            showDot={isCasting}
          />
        </button>

        {showSpotifyConnect && onConnectSpotify && (
          <button
            onClick={onConnectSpotify}
            className={`${btnClass} ml-auto gap-2 px-4 text-[#1DB954] hover:border-[#1DB954]/30 hover:bg-[#1DB954]/10`}
            title="Connect Spotify"
          >
            <SpotifyIcon className="h-6 w-6" />
            <span className="whitespace-nowrap font-display text-xs tracking-[0.2em]">
              Connect Spotify
            </span>
          </button>
        )}

        <button
          onClick={onAddSong}
          className={`${btnClass} ${!showSpotifyConnect ? 'ml-auto' : ''} gap-2 px-6 text-primary hover:border-primary/30`}
          title="Add Song"
        >
          <PlusIcon className="h-5 w-5 shrink-0" />
          <span className="whitespace-nowrap font-display text-theme text-xs tracking-[0.2em]">
            Add Song
          </span>
        </button>
      </div>
    </div>
  );
};

export const PlayerControls = React.memo(PlayerControlsComponent);
