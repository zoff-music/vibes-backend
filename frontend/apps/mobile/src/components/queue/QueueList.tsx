import React from 'react';
import { Song } from '@vibez/shared';
import { QueueItem } from './QueueItem';
import { Text } from '../ui/Text';

interface Props {
    songs: Song[];
    onRemove?: (id: string) => void;
    isAdmin?: boolean;
}

export const QueueList: React.FC<Props> = ({ songs, onRemove, isAdmin }) => {
    if (songs.length === 0) {
        return (
            <div className="py-10 flex items-center justify-center">
                <Text color="muted">The queue is empty. Add some vibes!</Text>
            </div>
        );
    }

    return (
        <div className="pb-5">
            {songs.map((song) => (
                <QueueItem
                    key={song.id}
                    song={song}
                    onRemove={onRemove}
                    isAdmin={isAdmin}
                />
            ))}
        </div>
    );
};
