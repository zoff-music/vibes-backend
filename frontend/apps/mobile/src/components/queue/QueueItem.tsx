import React from 'react';
import { Song } from '@vibez/shared';
import { Text } from '../ui/Text';

interface Props {
    song: Song;
    onRemove?: (id: string) => void;
    isAdmin?: boolean;
}

export const QueueItem: React.FC<Props> = ({ song, onRemove, isAdmin }) => {
    const formatDuration = (seconds: number) => {
        const min = Math.floor(seconds / 60);
        const sec = seconds % 60;
        return `${min}:${sec.toString().padStart(2, '0')}`;
    };

    return (
        <div className="flex items-center p-3 bg-surface rounded-xl mb-2">
            <img
                src={song.thumbnailUrl}
                alt={song.title}
                className="w-12 h-12 rounded-md object-cover bg-zinc-800"
            />

            <div className="flex-1 ml-3 min-w-0">
                <Text bold className="truncate">{song.title}</Text>
                <Text size="sm" color="muted" className="truncate">
                    {song.artist || 'Unknown Artist'} • {formatDuration(song.duration)}
                </Text>
            </div>

            <div className="flex items-center ml-3">
                {isAdmin && (
                    <button
                        onClick={() => onRemove?.(song.id)}
                        className="p-2 text-error hover:opacity-80 transition-opacity"
                    >
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            className="h-5 w-5"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                        >
                            <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                            />
                        </svg>
                    </button>
                )}
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="h-5 w-5 text-text-muted"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M4 6h16M4 12h16M4 18h16"
                    />
                </svg>
            </div>
        </div>
    );
};
