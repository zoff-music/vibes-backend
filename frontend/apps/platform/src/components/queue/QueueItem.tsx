import { Song } from '@vibez/shared';
import { motion } from 'framer-motion';
import React from 'react';

interface Props {
  song: Song;
  position: number;
  onRemove?: (id: string) => void;
  onVote?: (id: string) => void;
  isAdmin?: boolean;
}

export const QueueItem: React.FC<Props> = ({
  song,
  position,
  onRemove,
  onVote,
  isAdmin,
}) => {
  const formatDuration = (seconds: number) => {
    const min = Math.floor(seconds / 60);
    const sec = seconds % 60;
    return `${min}:${sec.toString().padStart(2, '0')}`;
  };

  return (
    <motion.div
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
      className="group glass cursor-pointer rounded-2xl border-2 border-ink/10 bg-white/50 p-4 backdrop-blur-sm transition-shadow hover:shadow-retro dark:border-primary/15 dark:bg-dark-surface/50"
    >
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
          <h4 className="mb-1 truncate font-bold text-ink transition-colors group-hover:text-primary dark:text-dark-text">
            {song.title}
          </h4>
          <div className="flex items-center gap-2 font-medium text-ink/60 text-sm dark:text-dark-text-muted">
            <span className="truncate">{song.artist || 'Unknown Artist'}</span>
            <span className="text-ink/40 dark:text-dark-text-subtle">•</span>
            <span className="shrink-0 font-mono text-xs">
              {formatDuration(song.duration)}
            </span>
            {(song.voteCount || 0) > 0 && (
              <>
                <span className="text-ink/40 dark:text-dark-text-subtle">
                  •
                </span>
                <span className="flex items-center gap-1 font-bold text-primary text-xs">
                  <svg
                    className="h-3 w-3"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v7.333l-2.62 1.53M6 14a1 1 0 011-1h1v1H7a1 1 0 01-1-1z" />
                  </svg>
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
              <svg
                className="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          )}
        </div>
      </div>
    </motion.div>
  );
};
