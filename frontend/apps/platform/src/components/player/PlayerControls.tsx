import React, { useState } from 'react';
import { useCasting } from '../../hooks/useCasting';
import { usePlayback } from '../../hooks/usePlayback';
import { DeviceSelector } from '../cast/DeviceSelector';

interface Props {
  roomId: string;
  hasSongsInQueue?: boolean;
  onAddSong: () => void;
  showSpotifyConnect?: boolean;
  onConnectSpotify?: () => void;
}

const PlayerControlsComponent: React.FC<Props> = ({
  roomId,
  hasSongsInQueue = false,
  onAddSong,
  showSpotifyConnect,
  onConnectSpotify,
}) => {
  const { isPlaying, play, pause, skip, currentSong } = usePlayback(roomId);
  const { isConnected, castDeviceName } = useCasting(roomId);
  const [showDeviceSelector, setShowDeviceSelector] = useState(false);

  const btnClass =
    'glass p-4 cursor-pointer rounded-xl hover:shadow-retro active:scale-95 transition-all disabled:opacity-30 disabled:cursor-not-allowed group border-2 border-ink/10 dark:border-primary/20 flex items-center justify-center h-14';

  return (
    <div className="w-full">
      {/* Main Controls */}
      <div className="flex items-center justify-start gap-4">
        {/* Play/Pause Button */}
        <button
          onClick={isPlaying ? pause : play}
          disabled={!currentSong && !hasSongsInQueue}
          className={`group flex h-14 w-14 cursor-pointer items-center justify-center rounded-xl border-2 border-white/50 bg-primary text-white shadow-retro-pink transition-all hover:shadow-neon-pink hover:shadow-retro active:scale-95 disabled:cursor-not-allowed disabled:opacity-30`}
        >
          {isPlaying ? (
            <svg className="h-6 w-6 fill-current" viewBox="0 0 24 24">
              <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
            </svg>
          ) : (
            <svg className="ml-0.5 h-6 w-6 fill-current" viewBox="0 0 24 24">
              <path d="M8 5v14l11-7z" />
            </svg>
          )}
        </button>

        {/* Skip Button */}
        <button
          onClick={skip}
          disabled={!currentSong}
          className={`${btnClass} w-14`}
          title="Skip"
        >
          <svg
            className="h-6 w-6 text-ink/60 transition-colors group-hover:text-primary dark:text-dark-text-muted dark:group-hover:text-primary"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2.5}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M13 5l7 7-7 7M5 5l7 7-7 7"
            />
          </svg>
        </button>

        {/* Cast Button */}
        <button
          onClick={() => setShowDeviceSelector(true)}
          className={`${btnClass} w-14 ${isConnected ? 'border-primary/20 bg-primary/10 dark:border-primary/30 dark:bg-primary/20' : ''}`}
          title={isConnected ? `Casting to ${castDeviceName}` : 'Cast'}
        >
          <svg
            className={`h-6 w-6 transition-colors ${isConnected ? 'text-primary dark:text-primary-light' : 'text-ink/60 group-hover:text-primary dark:text-dark-text-muted dark:group-hover:text-primary'}`}
            fill="currentColor"
            viewBox="0 0 24 24"
          >
            <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z" />
            {isConnected && (
              <circle cx="6" cy="18" r="2" className="fill-current" />
            )}
          </svg>
        </button>

        {/* Connect Spotify Button - Proactive Auth */}
        {showSpotifyConnect && onConnectSpotify && (
          <button
            onClick={onConnectSpotify}
            className={`${btnClass} ml-auto gap-2 px-4 text-[#1DB954] hover:border-[#1DB954]/30 hover:bg-[#1DB954]/10 dark:hover:bg-[#1DB954]/20`}
            title="Connect Spotify"
          >
            <svg className="h-6 w-6" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.4 0 0 5.4 0 12s5.4 12 12 12 12-5.4 12-12S18.66 0 12 0zm5.521 17.34c-.24.359-.66.48-1.021.24-2.82-1.74-6.36-2.101-10.561-1.141-.418.122-.779-.179-.899-.539-.12-.421.18-.78.54-.9 4.56-1.021 8.52-.6 11.64 1.32.42.18.479.659.301 1.02zm1.44-3.3c-.301.42-.841.6-1.262.3-3.239-1.98-8.159-2.58-11.939-1.38-.479.12-1.02-.12-1.14-.6-.12-.48.12-1.021.6-1.141C9.6 9.9 15 10.561 18.72 12.84c.361.181.54.78.241 1.2zm.12-3.36C15.24 8.4 8.82 8.16 5.16 9.301c-.6.179-1.2-.181-1.38-.721-.18-.601.18-1.2.72-1.381 4.26-1.26 11.28-1.02 15.721 1.621.539.3.719 1.02.419 1.56-.299.421-1.02.599-1.559.3z" />
            </svg>
            <span className="whitespace-nowrap font-black text-sm tracking-wide">
              Connect Spotify
            </span>
          </button>
        )}

        {/* Add Song Button */}
        <button
          onClick={onAddSong}
          className={`${btnClass} ${!showSpotifyConnect ? 'ml-auto' : ''} gap-2 px-6 text-primary hover:border-primary/30 dark:text-primary-light dark:hover:border-primary/40`}
          title="Add Song"
        >
          <svg
            className="h-5 w-5 shrink-0"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={3}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M12 4v16m8-8H4"
            />
          </svg>
          <span className="whitespace-nowrap font-black text-ink text-sm tracking-wide dark:text-dark-text">
            Add Song
          </span>
        </button>
      </div>

      {/* Casting Status */}
      {isConnected && castDeviceName && (
        <div className="mt-3 flex items-center justify-center gap-2 text-primary text-xs dark:text-primary-light">
          <div className="h-2 w-2 animate-pulse rounded-full bg-primary dark:bg-primary-light"></div>
          <span className="font-medium">Casting to {castDeviceName}</span>
        </div>
      )}

      {/* Device Selector Modal */}
      <DeviceSelector
        isOpen={showDeviceSelector}
        onClose={() => setShowDeviceSelector(false)}
      />
    </div>
  );
};

export const PlayerControls = React.memo(PlayerControlsComponent);
