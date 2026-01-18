import React from 'react';
import { Song } from '@vibez/shared';
import { motion } from 'framer-motion';

interface Props {
    song: Song;
    position: number;
    onRemove?: (id: string) => void;
    onVote?: (id: string) => void;
    isAdmin?: boolean;
}

export const QueueItem: React.FC<Props> = ({ song, position, onRemove, onVote, isAdmin }) => {
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
                type: "spring",
                stiffness: 500,
                damping: 30,
                mass: 1,
                opacity: { duration: 0.2 }
            }}
            onClick={() => onVote?.(song.id)}
            className="group glass rounded-2xl p-4 hover:shadow-retro transition-shadow border-2 border-ink/10 dark:border-primary/15 bg-white/50 dark:bg-dark-surface/50 backdrop-blur-sm cursor-pointer"
        >
            <div className="flex items-center gap-4">
                {/* Position number */}
                <div className="flex-shrink-0 w-8 text-center">
                    <span className="text-ink/50 dark:text-dark-text-subtle font-black text-lg">{position}</span>
                </div>

                {/* Thumbnail */}
                <div className="flex-shrink-0 relative">
                    <img
                        src={song.thumbnailUrl}
                        alt={song.title}
                        className="w-16 h-16 rounded-xl object-cover bg-surface dark:bg-dark-surfaceElevated ring-2 ring-ink/20 dark:ring-primary/20"
                        loading="lazy"
                    />
                </div>

                {/* Song info */}
                <div className="flex-1 min-w-0">
                    <h4 className="font-bold truncate mb-1 group-hover:text-primary transition-colors text-ink dark:text-dark-text">
                        {song.title}
                    </h4>
                    <div className="flex items-center gap-2 text-sm text-ink/60 dark:text-dark-text-muted font-medium">
                        <span className="truncate">{song.artist || 'Unknown Artist'}</span>
                        <span className="text-ink/40 dark:text-dark-text-subtle">•</span>
                        <span className="flex-shrink-0 font-mono text-xs">{formatDuration(song.duration)}</span>
                        {(song.voteCount || 0) > 0 && (
                            <>
                                <span className="text-ink/40 dark:text-dark-text-subtle">•</span>
                                <span className="flex items-center gap-1 text-xs text-primary font-bold">
                                    <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                                        <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v7.333l-2.62 1.53M6 14a1 1 0 011-1h1v1H7a1 1 0 01-1-1z" />
                                    </svg>
                                    {song.voteCount}
                                </span>
                            </>
                        )}
                    </div>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-1 flex-shrink-0">
                    {isAdmin && (
                        <button
                            onClick={(e) => {
                                e.stopPropagation();
                                onRemove?.(song.id);
                            }}
                            className="p-2.5 text-ink/40 dark:text-dark-text-subtle hover:text-error hover:bg-error/10 rounded-lg transition-all border-2 border-transparent hover:border-error/20"
                            title="Remove from queue"
                        >
                            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                        </button>
                    )}
                </div>
            </div>
        </motion.div>
    );
};
