import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useRoom } from '../hooks/useRoom';
import { usePlayback } from '../hooks/usePlayback';
import { useQueue } from '../hooks/useQueue';
import { VideoPlayer } from '../components/player/VideoPlayer';
import { PlayerControls } from '../components/player/PlayerControls';
import { AddToQueueModal } from '../components/queue/AddToQueueModal';
import { QueueList } from '../components/queue/QueueList';

export default function RoomView() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { currentSong, skip, isPlaying } = usePlayback(id || '');
    const { songs, fetchQueue } = useQueue(id || '');
    const { room, fetchRoom, isLoading, error, joinRoom, userId, users } = useRoom(id || '');

    const [isAddModalVisible, setIsAddModalVisible] = useState(false);
    const [showRoomInfo, setShowRoomInfo] = useState(false);

    useEffect(() => {
        if (id) {
            fetchRoom();
            fetchQueue();
        }
    }, [id, fetchRoom, fetchQueue]);

    useEffect(() => {
        if (!id || userId) return;

        const checkJoin = async () => {
            await joinRoom("Guest_" + Math.floor(Math.random() * 1000));
        };

        checkJoin();
    }, [id, userId, joinRoom]);

    if (isLoading && !room) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="text-center animate-fade-in">
                    <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl glass-elevated mb-4">
                        <div className="w-8 h-8 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
                    </div>
                    <p className="text-text-muted">Loading session...</p>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen flex flex-col items-center justify-center px-4">
                <div className="max-w-md w-full text-center animate-scale-in">
                    <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl glass-elevated mb-4">
                        <svg className="w-8 h-8 text-error" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                    </div>
                    <h2 className="text-xl font-bold mb-2">Connection Failed</h2>
                    <p className="text-text-muted mb-6">{error.message}</p>
                    <button
                        onClick={() => fetchRoom()}
                        className="glass-elevated px-6 py-3 rounded-lg hover:bg-surfaceHover transition-colors"
                    >
                        Try Again
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen flex flex-col animate-fade-in">
            {/* Header */}
            <div className="px-4 py-6 backdrop-blur-xl bg-background/80 border-b border-border sticky top-0 z-20">
                <div className="max-w-7xl mx-auto flex items-center justify-between">
                    <button
                        onClick={() => navigate(-1)}
                        className="inline-flex items-center gap-2 text-text-muted hover:text-white transition-colors group"
                    >
                        <svg className="w-5 h-5 group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                        </svg>
                        <span className="text-sm font-medium">Leave</span>
                    </button>

                    <button
                        onClick={() => setShowRoomInfo(!showRoomInfo)}
                        className="flex-1 mx-4 flex items-center justify-center gap-2 hover:opacity-80 transition-opacity"
                    >
                        <h1 className="text-lg font-semibold truncate">{room?.name || 'Loading...'}</h1>
                        <svg className={`w-4 h-4 text-text-subtle transition-transform ${showRoomInfo ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                        </svg>
                    </button>

                    <div className="flex items-center gap-1">
                        {users && users.length > 0 && (
                            <div className="flex items-center gap-1.5 glass rounded-full px-3 py-1.5">
                                <div className="w-2 h-2 bg-success rounded-full animate-pulse" />
                                <span className="text-xs font-medium">{users.length}</span>
                            </div>
                        )}
                    </div>
                </div>

                {/* Room info dropdown */}
                {showRoomInfo && (
                    <div className="mt-4 glass-elevated rounded-xl p-4 animate-slide-down">
                        <div className="space-y-3">
                            <div>
                                <p className="text-xs text-text-subtle uppercase tracking-wider mb-1">Room Code</p>
                                <div className="flex items-center gap-2">
                                    <code className="text-sm font-mono text-white bg-surfaceElevated px-3 py-1.5 rounded-lg">{id}</code>
                                    <button
                                        onClick={() => navigator.clipboard.writeText(id || '')}
                                        className="p-1.5 hover:bg-surfaceHover rounded-lg transition-colors"
                                        title="Copy code"
                                    >
                                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                        </svg>
                                    </button>
                                </div>
                            </div>
                            {users && users.length > 0 && (
                                <div>
                                    <p className="text-xs text-text-subtle uppercase tracking-wider mb-2">Listeners ({users.length})</p>
                                    <div className="flex flex-wrap gap-2">
                                        {users.slice(0, 8).map((user) => (
                                            <div key={user.id} className="text-xs bg-surfaceElevated px-2.5 py-1 rounded-full text-text-muted">
                                                {user.nickname || `User ${user.id.slice(0, 4)}`}
                                            </div>
                                        ))}
                                        {users.length > 8 && (
                                            <div className="text-xs bg-surfaceElevated px-2.5 py-1 rounded-full text-text-subtle">
                                                +{users.length - 8} more
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                )}
            </div>

            {/* Main content */}
            <div className="flex-1 overflow-y-auto">
                <div className="max-w-7xl mx-auto px-4 py-8 space-y-8">
                    {/* Player Section */}
                    <div className="space-y-6">
                        {/* Video Player - Hidden but functional */}
                        <div className="hidden">
                            <VideoPlayer onEnded={skip} />
                        </div>

                        {/* Now Playing Card */}
                        <div className="glass-elevated rounded-2xl p-8 relative overflow-hidden animate-scale-in">
                            {/* Playing indicator glow */}
                            {isPlaying && (
                                <div className="absolute inset-0 bg-gradient-to-br from-primary/10 via-transparent to-secondary/10 animate-glow-pulse pointer-events-none" />
                            )}

                            <div className="relative">
                                <div className="flex items-center gap-2 mb-6">
                                    <div className={`w-2 h-2 rounded-full ${isPlaying ? 'bg-success animate-pulse' : 'bg-text-subtle'}`} />
                                    <span className="text-xs text-text-subtle uppercase tracking-wider">
                                        {isPlaying ? 'Now Playing' : 'Paused'}
                                    </span>
                                </div>

                                {currentSong ? (
                                    <div className="text-center mb-6">
                                        <h2 className="text-3xl font-bold mb-2 line-clamp-2">
                                            {currentSong.title}
                                        </h2>
                                        <p className="text-lg text-text-muted">
                                            {currentSong.artist || 'Unknown Artist'}
                                        </p>
                                    </div>
                                ) : (
                                    <div className="text-center mb-6 py-8">
                                        <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-surfaceElevated mb-4">
                                            <svg className="w-8 h-8 text-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
                                            </svg>
                                        </div>
                                        <h3 className="text-xl font-semibold mb-2">No song playing</h3>
                                        <p className="text-text-muted">Add a song to get started</p>
                                    </div>
                                )}

                                {/* Player Controls */}
                                <PlayerControls roomId={id || ''} />
                            </div>
                        </div>
                    </div>

                    {/* Queue Section */}
                    <div>
                        <div className="flex items-center justify-between mb-4">
                            <div>
                                <h2 className="text-xl font-bold">Up Next</h2>
                                <p className="text-sm text-text-muted mt-0.5">
                                    {songs.length} {songs.length === 1 ? 'song' : 'songs'} in queue
                                </p>
                            </div>
                            <button
                                onClick={() => setIsAddModalVisible(true)}
                                className="glass-elevated px-5 py-2.5 rounded-xl hover:bg-surfaceHover transition-all hover:scale-105 active:scale-95 flex items-center gap-2 group"
                            >
                                <svg className="w-5 h-5 text-primary group-hover:rotate-90 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                                </svg>
                                <span className="font-medium">Add Song</span>
                            </button>
                        </div>

                        <QueueList songs={songs} roomId={id || ''} />
                    </div>
                </div>
            </div>

            {/* Add Song Modal */}
            <AddToQueueModal
                roomId={id || ''}
                isVisible={isAddModalVisible}
                onClose={() => setIsAddModalVisible(false)}
            />
        </div>
    );
}
