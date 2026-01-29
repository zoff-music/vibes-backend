import { Song } from '@vibez/shared';
import { motion } from 'framer-motion';
import React from 'react';
import {
  SoundCloudIcon,
  SpotifyIcon,
  TrashIcon,
  VoteIcon,
  YouTubeIcon,
} from '../../icons';

interface Props {
  song: Song;
  position: number;
  onRemove?: (id: string) => void;
  onVote?: (id: string) => void;
  isAdmin?: boolean;
  isSSR?: boolean;
}

export const QueueItem: React.FC<Props> = ({
  song,
  position,
  onRemove,
  onVote,
  isAdmin,
  isSSR = false,
}) => {
  const formatDuration = (seconds: number) => {
    const min = Math.floor(seconds / 60);
    const sec = seconds % 60;
    return `${min}:${sec.toString().padStart(2, '0')}`;
  };

  const content = (
    <div className="flex items-center gap-4">
      {/* Position number */}
      <div className="w-8 shrink-0 text-center">
        <span className="font-display text-theme-subtle text-xs">
          {position}
        </span>
      </div>

      {/* Thumbnail */}
      <div className="relative shrink-0">
        <img
          src={song.thumbnailUrl}
          alt={song.title}
          className="h-16 w-16 rounded-xl border border-theme bg-theme-surface object-cover"
          loading="lazy"
        />
      </div>

      {/* Song info */}
      <div className="min-w-0 flex-1">
        <h4 className="mb-1 truncate text-left font-display text-theme text-xs">
          {song.title}
        </h4>
        <div className="flex items-center gap-2 text-theme-muted text-xs">
          <span className="truncate">{song.artist || 'Unknown Artist'}</span>
          <span className="text-theme-subtle">•</span>
          <span className="shrink-0 font-mono text-theme-subtle text-xs">
            {formatDuration(song.duration)}
          </span>
          {(song.voteCount || 0) > 0 && (
            <>
              <span className="text-theme-subtle">•</span>
              <span className="flex items-center gap-1 font-display text-[10px] text-secondary">
                <VoteIcon className="h-3 w-3" />
                {song.voteCount}
              </span>
            </>
          )}
        </div>
      </div>

      {/* Actions */}
      <div className="flex shrink-0 items-center gap-3 pr-4">
        {/* Source Icon */}
        <div className="flex items-center justify-center opacity-70">
          {song.sourceType === 'spotify' ? (
            <SpotifyIcon className="h-5 w-5" />
          ) : song.sourceType === 'soundcloud' ? (
            <SoundCloudIcon className="h-5 w-5" />
          ) : (
            <YouTubeIcon className="h-5 w-5" />
          )}
        </div>

        {isAdmin && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onRemove?.(song.id);
            }}
            className="cursor-pointer rounded-lg border border-transparent p-2.5 text-theme-subtle transition-all hover:border-error/40 hover:bg-error/10 hover:text-error"
            title="Remove from queue"
          >
            <TrashIcon className="h-5 w-5" />
          </button>
        )}
      </div>
    </div>
  );

  if (isSSR) {
    // SSR: Render without motion.div
    return (
      <button
        type="button"
        onClick={() => onVote?.(song.id)}
        className="group w-full cursor-pointer rounded-2xl border border-theme bg-theme-surface p-4 transition-shadow hover:shadow-[0_0_20px_rgba(255,46,151,0.2)] focus:outline-none focus:ring-2 focus:ring-secondary/40 focus:ring-offset-2 focus:ring-offset-transparent"
        aria-label={`Vote for ${song.title} by ${song.artist || 'Unknown Artist'}`}
      >
        {content}
      </button>
    );
  }

  // Client: Render with motion.div - using motion.button for semantic consistency
  return (
    <motion.button
      type="button"
      layout
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.95, transition: { duration: 0.2 } }}
      transition={{
        type: 'spring',
        stiffness: 500,
        damping: 30,
        mass: 1,
        opacity: { duration: 0.2 },
      }}
      onClick={() => onVote?.(song.id)}
      className="group w-full cursor-pointer rounded-2xl border border-theme bg-theme-surface p-4 transition-shadow hover:shadow-[0_0_20px_rgba(255,46,151,0.2)] focus:outline-none focus:ring-2 focus:ring-secondary/40 focus:ring-offset-2 focus:ring-offset-transparent"
      aria-label={`Vote for ${song.title} by ${song.artist || 'Unknown Artist'}`}
    >
      {content}
    </motion.button>
  );
};
