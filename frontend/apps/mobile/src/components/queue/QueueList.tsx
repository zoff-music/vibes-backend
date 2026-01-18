import React from 'react';
import { Song } from '@vibez/shared';
import { QueueItem } from './QueueItem';

interface Props {
    songs: Song[];
    roomId?: string; // eslint-disable-line @typescript-eslint/no-unused-vars
    onRemove?: (id: string) => void;
    isAdmin?: boolean;
}

export const QueueList: React.FC<Props> = ({ songs, roomId, onRemove, isAdmin }) => {
    if (songs.length === 0) {
        return (
            <div className="glass rounded-2xl p-12 text-center animate-fade-in">
                <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-surfaceElevated mb-4">
                    <svg className="w-8 h-8 text-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                </div>
                <h3 className="text-lg font-semibold mb-2">Queue is Empty</h3>
                <p className="text-text-muted">Add some songs to get the party started</p>
            </div>
        );
    }

    return (
        <div className="space-y-3">
            {songs.map((song, index) => (
                <QueueItem
                    key={song.id}
                    song={song}
                    position={index + 1}
                    onRemove={onRemove}
                    isAdmin={isAdmin}
                />
            ))}
        </div>
    );
};
