import React from 'react';
import { Song } from '@vibez/shared';

interface Props {
    song: Song;
    position: number;
    onRemove?: (id: string) => void;
    isAdmin?: boolean;
}

export const QueueItem: React.FC<Props> = ({ song, position, onRemove, isAdmin }) => {
    const formatDuration = (seconds: number) => {
        const min = Math.floor(seconds / 60);
        const sec = seconds % 60;
        return `${min}:${sec.toString().padStart(2, '0')}`;
    };

    return (
        <div
            className="group glass rounded-xl p-4 hover:bg-surfaceHover transition-all animate-slide-up"
            style={{ animationDelay: `${position * 0.05}s` }}
        >
            <div className="flex items-center gap-4">
                {/* Position number */}
                <div className="flex-shrink-0 w-8 text-center">
                    <span className="text-text-subtle font-medium">{position}</span>
                </div>

                {/* Thumbnail */}
                <div className="flex-shrink-0 relative">
                    <img
                        src={song.thumbnailUrl}
                        alt={song.title}
                        className="w-14 h-14 rounded-lg object-cover bg-surfaceElevated ring-1 ring-border"
                        loading="lazy"
                    />
                    <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent rounded-lg pointer-events-none" />
                </div>

                {/* Song info */}
                <div className="flex-1 min-w-0">
                    <h4 className="font-medium truncate mb-0.5 group-hover:text-primary transition-colors">
                        {song.title}
                    </h4>
                    <div className="flex items-center gap-2 text-sm text-text-muted">
                        <span className="truncate">{song.artist || 'Unknown Artist'}</span>
                        <span className="text-text-subtle">•</span>
                        <span className="flex-shrink-0">{formatDuration(song.duration)}</span>
                    </div>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-1 flex-shrink-0">
                    {isAdmin && (
                        <button
                            onClick={() => onRemove?.(song.id)}
                            className="p-2 text-text-subtle hover:text-error hover:bg-error/10 rounded-lg transition-all"
                            title="Remove from queue"
                        >
                            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                        </button>
                    )}
                </div>
            </div>
        </div>
    );
};
