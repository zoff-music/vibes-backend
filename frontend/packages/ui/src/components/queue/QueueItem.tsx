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

const vinylPlaceholder =
  'data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%27http%3A//www.w3.org/2000/svg%27%20width%3D%27200%27%20height%3D%27200%27%20viewBox%3D%270%200%20200%20200%27%3E%3Crect%20width%3D%27200%27%20height%3D%27200%27%20rx%3D%2724%27%20fill%3D%27%2316161c%27/%3E%3Ccircle%20cx%3D%27100%27%20cy%3D%27100%27%20r%3D%2772%27%20fill%3D%27%231f1f27%27/%3E%3Ccircle%20cx%3D%27100%27%20cy%3D%27100%27%20r%3D%2752%27%20fill%3D%27%232a2a34%27/%3E%3Ccircle%20cx%3D%27100%27%20cy%3D%27100%27%20r%3D%2718%27%20fill%3D%27%23141418%27/%3E%3Ccircle%20cx%3D%27100%27%20cy%3D%27100%27%20r%3D%276%27%20fill%3D%27%23c7c7d1%27/%3E%3Cpath%20d%3D%27M100%2028a72%2072%200%200%201%2072%2072%27%20stroke%3D%27%233a3a46%27%20stroke-width%3D%276%27%20fill%3D%27none%27%20stroke-linecap%3D%27round%27/%3E%3C/svg%3E';

const resolveThumbnail = (value?: string) =>
  value && value.trim().length > 0 ? value : vinylPlaceholder;

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
        <span className="text-theme-subtle text-xs">{position}</span>
      </div>

      {/* Thumbnail */}
      <div className="relative shrink-0">
        <img
          src={resolveThumbnail(song.thumbnailUrl)}
          alt={song.title}
          className="h-16 w-16 rounded-xl border border-theme bg-theme-surface object-cover"
          loading="lazy"
        />
      </div>

      {/* Song info */}
      <div className="min-w-0 flex-1">
        <h4 className="mb-1 truncate text-left text-theme text-xs">
          {song.title}
        </h4>
        <div className="flex items-center gap-2 text-theme-muted text-xs">
          <span className="truncate">{song.artist || 'Unknown Artist'}</span>
          <span className="text-theme-subtle">•</span>
          <span className="shrink-0 text-theme-subtle text-xs">
            {formatDuration(song.duration)}
          </span>
          {(song.voteCount || 0) > 0 && (
            <>
              <span className="text-theme-subtle">•</span>
              <span className="flex items-center gap-1 text-[10px] text-secondary">
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

  // Client: Render with motion.div wrapper for height animation
  return (
    <motion.div
      layout
      initial={{ opacity: 0, height: 0, y: 50 }}
      animate={{ opacity: 1, height: 'auto', y: 0 }}
      exit={{ opacity: 0, height: 0, y: -50, transition: { duration: 0.2 } }}
      transition={{
        type: 'spring',
        stiffness: 500,
        damping: 35,
        opacity: { duration: 0.15 },
      }}
      style={{ overflow: 'hidden' }}
    >
      <button
        type="button"
        onClick={() => onVote?.(song.id)}
        className="group w-full cursor-pointer rounded-2xl border border-theme bg-theme-surface p-4 transition-shadow hover:shadow-[0_0_20px_rgba(255,46,151,0.2)] focus:outline-none focus:ring-2 focus:ring-secondary/40 focus:ring-offset-2 focus:ring-offset-transparent"
        aria-label={`Vote for ${song.title} by ${song.artist || 'Unknown Artist'}`}
      >
        {content}
      </button>
    </motion.div>
  );
};
