import React from 'react';
import { Song } from '@vibez/shared';
import { QueueItem } from './QueueItem';
import { AnimatePresence } from 'framer-motion';

interface Props {
    songs: Song[];
    roomId?: string; // eslint-disable-line @typescript-eslint/no-unused-vars
    onRemove?: (id: string) => void;
    onVote?: (id: string) => void;
    isAdmin?: boolean;
}

export const QueueList: React.FC<Props> = ({ songs, onRemove, onVote, isAdmin }) => {
    if (songs.length === 0) {
        return (
            <div className="glass rounded-3xl p-12 text-center animate-fade-in border-2 border-ink/10 bg-white/50">
                <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-white border-4 border-ink/20 shadow-retro mb-5">
                    <svg className="w-10 h-10 text-ink/40" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                </div>
                <h3 className="text-xl font-black mb-2 text-ink" style={{ fontFamily: 'Poppins' }}>Queue is Empty</h3>
                <p className="text-ink/60 font-medium mb-2">Add some songs to get the party started</p>
                <p className="text-sm jp-art text-ink/40">曲を追加</p>
            </div>
        );
    }

    return (
        <div className="space-y-3">
            <AnimatePresence initial={false} mode="popLayout">
                {songs
                    .filter(song => song.position > 0) // Filter out current playing song (position 0)
                    .map((song, index) => (
                    <QueueItem
                        key={song.id}
                        song={song}
                        position={index + 1}
                        onRemove={onRemove}
                        onVote={onVote}
                        isAdmin={isAdmin}
                    />
                ))}
            </AnimatePresence>
        </div>
    );
};
