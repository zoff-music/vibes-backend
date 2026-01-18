import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useRoom } from '../hooks/useRoom';
import { usePlayback } from '../hooks/usePlayback';
import { useQueue } from '../hooks/useQueue';
import { VideoPlayer } from '../components/player/VideoPlayer';
import { PlayerControls } from '../components/player/PlayerControls';
import { AddToQueueModal } from '../components/queue/AddToQueueModal';
import { Button } from '../components/ui/Button';

export default function RoomView() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { currentSong, skip } = usePlayback(id || '');
    const { songs, fetchQueue } = useQueue(id || '');
    const { room, fetchRoom, isLoading, error, joinRoom, userId } = useRoom(id || '');

    const [isAddModalVisible, setIsAddModalVisible] = useState(false);

    useEffect(() => {
        if (id) {
            fetchRoom();
            fetchQueue();
        }
    }, [id, fetchRoom, fetchQueue]);

    // Auto-join if not already in session
    useEffect(() => {
        if (!id || userId) return;

        const checkJoin = async () => {
            // For now, just join as "Guest"
            await joinRoom("Guest_" + Math.floor(Math.random() * 1000));
        };

        checkJoin();
    }, [id, userId, joinRoom]);

    if (isLoading && !room) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <p className="text-text">Loading Room...</p>
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen bg-background flex flex-col items-center justify-center">
                <p className="text-primary">Error: {error.message}</p>
                <Button onClick={() => fetchRoom()} title="Retry" className="mt-5" />
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-background pt-12 flex flex-col">
            <div className="px-6 flex items-center justify-between mb-6">
                <button
                    onClick={() => navigate(-1)}
                    className="text-primary text-base hover:opacity-80"
                >
                    ← Back
                </button>
                <h1 className="text-xl font-bold text-text">{room?.name || 'Loading...'}</h1>
                <div className="w-12" /> {/* Spacer for centering */}
            </div>

            <div className="flex-1 overflow-y-auto">
                <div className="px-6 mb-6">
                    <div className="mb-6">
                        <VideoPlayer onEnded={skip} />

                        <div className="mt-4 text-center">
                            <h2 className="text-2xl font-bold text-text line-clamp-1">
                                {currentSong?.title || 'No song playing'}
                            </h2>
                            <p className="text-text-muted text-base mt-1">
                                {currentSong?.artist || 'Add songs to get started'}
                            </p>
                        </div>
                    </div>
                </div>

                <PlayerControls roomId={id || ''} />

                <div className="px-6 mt-8 mb-4 flex items-center justify-between">
                    <h2 className="text-lg font-bold text-text">Up Next</h2>
                    <Button
                        onClick={() => setIsAddModalVisible(true)}
                        title="Add Song"
                        variant="ghost"
                        size="sm"
                    />
                </div>

                <div className="px-6 pb-10">
                    {songs.length === 0 ? (
                        <p className="text-text-muted text-base text-center py-8">Queue is empty</p>
                    ) : (
                        <div className="space-y-2">
                            {songs.map((song, index) => (
                                <div key={song.id} className="flex items-center bg-surface p-3 rounded-lg">
                                    <div className="w-8 text-text-muted text-sm">{index + 1}</div>
                                    <div className="flex-1 min-w-0">
                                        <p className="text-text truncate">{song.title}</p>
                                        <p className="text-text-muted text-xs mt-0.5">
                                            {song.artist || 'Unknown Artist'}
                                        </p>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>

            <div className="py-4 text-center border-t border-surfaceElevated">
                <p className="text-text-muted text-xs">Room PIN: {id}</p>
            </div>

            <AddToQueueModal
                roomId={id || ''}
                isVisible={isAddModalVisible}
                onClose={() => setIsAddModalVisible(false)}
            />
        </div>
    );
}
