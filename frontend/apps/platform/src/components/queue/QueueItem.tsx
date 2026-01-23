import { Song } from '@vibez/shared';
import { TrashIcon, VoteIcon } from '@vibez/ui';
import { motion } from 'framer-motion';
import React from 'react';

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
        <span className="font-black text-ink/50 text-lg dark:text-dark-text-subtle">
          {position}
        </span>
      </div>

      {/* Thumbnail */}
      <div className="relative shrink-0">
        <img
          src={song.thumbnailUrl}
          alt={song.title}
          className="h-16 w-16 rounded-xl bg-surface object-cover ring-2 ring-ink/20 dark:bg-dark-surfaceElevated dark:ring-primary/20"
          loading="lazy"
        />
      </div>

      {/* Song info */}
      <div className="min-w-0 flex-1">
        <h4 className="mb-1 truncate text-left font-bold text-ink text-sm transition-colors group-hover:text-primary dark:text-dark-text">
          {song.title}
        </h4>
        <div className="flex items-center gap-2 font-medium text-ink/60 text-xs dark:text-dark-text-muted">
          <span className="truncate">{song.artist || 'Unknown Artist'}</span>
          <span className="text-ink/40 dark:text-dark-text-subtle">•</span>
          <span className="shrink-0 font-mono text-xs">
            {formatDuration(song.duration)}
          </span>
          {(song.voteCount || 0) > 0 && (
            <>
              <span className="text-ink/40 dark:text-dark-text-subtle">•</span>
              <span className="flex items-center gap-1 font-bold text-primary text-xs">
                <VoteIcon className="h-3 w-3" />
                {song.voteCount}
              </span>
            </>
          )}
        </div>
      </div>

      {/* Actions */}
      <div className="flex shrink-0 items-center gap-1">
        {isAdmin && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onRemove?.(song.id);
            }}
            className="cursor-pointer rounded-lg border-2 border-transparent p-2.5 text-ink/40 transition-all hover:border-error/20 hover:bg-error/10 hover:text-error dark:text-dark-text-subtle"
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
        className="group glass w-full cursor-pointer rounded-2xl border-2 border-ink/10 bg-white/50 p-4 backdrop-blur-sm transition-shadow hover:shadow-retro focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:border-primary/15 dark:bg-dark-surface/50 dark:focus:ring-offset-gray-800"
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
      className="group glass w-full cursor-pointer rounded-2xl border-2 border-ink/10 bg-white/50 p-4 backdrop-blur-sm transition-shadow hover:shadow-retro focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:border-primary/15 dark:bg-dark-surface/50 dark:focus:ring-offset-gray-800"
      aria-label={`Vote for ${song.title} by ${song.artist || 'Unknown Artist'}`}
    >
      {content}
    </motion.button>
  );
};
