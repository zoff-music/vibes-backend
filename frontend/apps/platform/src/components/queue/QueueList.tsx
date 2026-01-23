import { Song } from '@vibez/shared';
import { AnimatePresence } from 'framer-motion';
import React, { useEffect, useState } from 'react';
import { QueueItem } from './QueueItem';

interface Props {
  songs: Song[];
  roomId?: string; // eslint-disable-line @typescript-eslint/no-unused-vars
  onRemove?: (id: string) => void;
  onVote?: (id: string) => void;
  isAdmin?: boolean;
}

const QueueListComponent: React.FC<Props> = ({
  songs,
  onRemove,
  onVote,
  isAdmin,
}) => {
  const [isSSR, setIsSSR] = useState(true);

  useEffect(() => {
    setIsSSR(false);
  }, []);

  if (songs.length === 0) {
    return (
      <div className="glass animate-fade-in rounded-3xl border-2 border-ink/10 bg-white/50 p-12 text-center dark:border-primary/20 dark:bg-dark-surface/60">
        <div className="mb-5 inline-flex h-20 w-20 items-center justify-center rounded-2xl border-4 border-ink/20 bg-white shadow-retro dark:border-primary/20 dark:bg-dark-surfaceElevated">
          <svg
            className="h-10 w-10 text-ink/40 dark:text-dark-text-muted"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
            />
          </svg>
        </div>
        <h3
          className="mb-2 font-black text-ink text-xl dark:text-dark-text"
          style={{ fontFamily: 'Poppins' }}
        >
          Queue is Empty
        </h3>
        <p className="mb-2 font-medium text-ink/60 dark:text-dark-text-muted">
          Add some songs to get the party started
        </p>
        <p className="jp-art text-ink/40 text-sm dark:text-dark-text-subtle">
          曲を追加
        </p>
      </div>
    );
  }

  const queueSongs = songs.filter((song) => song.position > 0); // Filter out current playing song (position 0)

  return (
    <div className="space-y-3">
      {isSSR ? (
        // SSR: Render without animations
        queueSongs.map((song, index) => (
          <QueueItem
            key={song.id}
            song={song}
            position={index + 1}
            onRemove={onRemove}
            onVote={onVote}
            isAdmin={isAdmin}
            isSSR={true}
          />
        ))
      ) : (
        // Client: Render with animations
        <AnimatePresence initial={false} mode="popLayout">
          {queueSongs.map((song, index) => (
            <QueueItem
              key={song.id}
              song={song}
              position={index + 1}
              onRemove={onRemove}
              onVote={onVote}
              isAdmin={isAdmin}
              isSSR={false}
            />
          ))}
        </AnimatePresence>
      )}
    </div>
  );
};

export const QueueList = React.memo(QueueListComponent);
