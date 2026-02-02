import { useQueue } from '@vibez/api';
import { type PlaybackState, type Song } from '@vibez/models';
import { usePlaybackStore } from '@vibez/shared';
import { QueueList, SoundCloudIcon, SpotifyIcon, YouTubeIcon } from '@vibez/ui';
import { AnimatePresence, motion } from 'framer-motion';
import React from 'react';
import { PlaybackProgress } from './PlaybackProgress';

const vinylPlaceholder =
  'data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%27http%3A//www.w3.org/2000/svg%27%20width%3D%27200%27%20height%3D%27200%27%20viewBox%3D%270%200%20200%20200%27%3E%3Crect%20width%3D%27200%27%20height%3D%27200%27%20rx%3D%2724%27%20fill%3D%27%2316161c%27/%3E%3Ccircle%20cx%3D%27100%27%20cy%3D%27100%27%20r%3D%2772%27%20fill%3D%27%231f1f27%27/%3E%3Ccircle%20cx%3D%27100%27%20cy%3D%27100%27%20r%3D%2752%27%20fill%3D%27%232a2a34%27/%3E%3Ccircle%20cx%3D%27100%27%20cy%3D%27100%27%20r%3D%2718%27%20fill%3D%27%23141418%27/%3E%3Ccircle%20cx%3D%27100%27%20cy%3D%27100%27%20r%3D%276%27%20fill%3D%27%23c7c7d1%27/%3E%3Cpath%20d%3D%27M100%2028a72%2072%200%200%201%2072%2072%27%20stroke%3D%27%233a3a46%27%20stroke-width%3D%276%27%20fill%3D%27none%27%20stroke-linecap%3D%27round%27/%3E%3C/svg%3E';

const resolveThumbnail = (value?: string) =>
  value && value.trim().length > 0 ? value : vinylPlaceholder;

interface RoomQueueProps {
  roomId: string;
  isSSR: boolean;
  initialPlayback?: PlaybackState;
  initialSongs?: Song[];
}

export const RoomQueue: React.FC<RoomQueueProps> = React.memo(
  ({ roomId, isSSR, initialPlayback, initialSongs }: RoomQueueProps) => {
    /* 1. Hooks */
    const { songs, voteSong } = useQueue(roomId);

    // Granular store subscriptions
    const currentSongFromStore = usePlaybackStore((state) => state.currentSong);
    const isPlayingFromStore = usePlaybackStore((state) => state.isPlaying);

    /* 2. State & Computed */
    const currentSong =
      currentSongFromStore || initialPlayback?.currentSong || null;
    const isPlaying =
      isPlayingFromStore !== undefined
        ? isPlayingFromStore
        : initialPlayback?.isPlaying || false;
    const displaySongs =
      isSSR && initialSongs
        ? initialSongs
        : songs.length > 0
          ? songs
          : initialSongs || [];

    /* 3. Handlers */
    const handleVote = React.useCallback(
      async (songId: string) => {
        await voteSong(songId);
      },
      [voteSong],
    );

    const formatTime = (ms: number) => {
      const seconds = Math.floor(ms / 1000);
      const m = Math.floor(seconds / 60);
      const s = seconds % 60;
      return `${m}:${s.toString().padStart(2, '0')}`;
    };

    return (
      <div className="mt-8 space-y-8 lg:mt-0 lg:h-full lg:overflow-y-auto lg:pr-2">
        <div className="relative lg:pb-6">
          {/* Now Playing (Integrated into list style) */}
          {currentSong && (
            <div className="mb-8">
              <div className="mb-3 flex items-center gap-2">
                <div
                  className={`h-2 w-2 rounded-full ${
                    isPlaying
                      ? 'animate-pulse bg-secondary shadow-[0_0_10px_rgba(0,217,255,0.6)]'
                      : 'bg-white/30'
                  }`}
                />
                <span className="font-display text-[10px] text-theme-muted tracking-[0.3em]">
                  {isPlaying ? 'Now Playing' : 'Paused'}
                </span>
              </div>

              <AnimatePresence initial={false} mode="popLayout">
                <motion.div
                  key={currentSong.id}
                  layout
                  initial={{ opacity: 0, y: 50 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{
                    opacity: 0,
                    y: -50,
                    position: 'absolute',
                    width: '100%',
                    transition: { duration: 0.2 },
                  }}
                  transition={{
                    type: 'spring',
                    stiffness: 500,
                    damping: 35,
                    opacity: { duration: 0.15 },
                  }}
                  style={{ overflow: 'hidden' }}
                >
                  <div className="group/card panel-surface no-box relative flex items-center gap-4 overflow-hidden rounded-2xl p-4">
                    <div className="vhs-scanlines pointer-events-none absolute inset-0" />

                    {/* Thumbnail */}
                    <div className="relative z-10 shrink-0">
                      <img
                        src={resolveThumbnail(currentSong.thumbnailUrl)}
                        alt=""
                        className="h-16 w-16 rounded-xl border border-theme object-cover shadow-xs transition-transform group-hover/card:scale-105"
                      />
                    </div>

                    {/* Song info */}
                    <div className="relative z-10 min-w-0 flex-1">
                      <h3 className="mb-1 truncate font-display text-theme text-xs">
                        {currentSong.title}
                      </h3>
                      <div className="flex items-center gap-2 text-theme-muted text-xs">
                        <span className="truncate">
                          {currentSong.artist || 'Unknown Artist'}
                        </span>
                        <span className="text-theme-subtle">•</span>
                        <span className="shrink-0 font-mono text-theme-subtle text-xs">
                          {formatTime(currentSong.duration * 1000)}
                        </span>
                      </div>
                    </div>

                    {/* Source Icon */}
                    <div className="relative z-10 flex shrink-0 items-center justify-center opacity-70">
                      {currentSong.sourceType === 'spotify' ? (
                        <SpotifyIcon className="h-5 w-5" />
                      ) : currentSong.sourceType === 'soundcloud' ? (
                        <SoundCloudIcon className="h-5 w-5" />
                      ) : (
                        <YouTubeIcon className="h-5 w-5" />
                      )}
                    </div>
                  </div>
                </motion.div>
              </AnimatePresence>

              <PlaybackProgress
                durationMs={currentSong.duration * 1000}
                isSSR={isSSR}
              />

              <div className="mt-8 mb-4 h-px bg-theme-surface" />
            </div>
          )}

          {/* Up Next List */}
          <div>
            <h3 className="mb-4 font-display text-[10px] text-theme-muted tracking-[0.3em]">
              Up Next (
              {displaySongs.filter((s) => s.id !== currentSong?.id).length})
            </h3>
            <QueueList
              songs={displaySongs.filter((s) => s.id !== currentSong?.id)}
              roomId={roomId}
              onVote={handleVote}
            />
          </div>
        </div>
      </div>
    );
  },
);
