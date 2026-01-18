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
                    <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-white border-4 border-ink/20 shadow-retro mb-5">
                        <div className="w-8 h-8 border-3 border-primary/30 border-t-primary rounded-full animate-spin" />
                    </div>
                    <p className="text-ink/60 font-medium">Loading session...</p>
                    <p className="text-sm jp-art text-ink/40 mt-1">読み込み中</p>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen flex flex-col items-center justify-center px-4">
                <div className="max-w-md w-full text-center animate-scale-in">
                    <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-white border-4 border-error shadow-retro mb-5">
                        <svg className="w-10 h-10 text-error" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                    </div>
                    <h2 className="text-2xl font-black mb-2 text-ink" style={{ fontFamily: 'Poppins' }}>Connection Failed</h2>
                    <p className="text-ink/60 mb-6 font-medium">{error.message}</p>
                    <button
                        onClick={() => fetchRoom()}
                        className="glass-elevated px-8 py-3.5 rounded-xl hover:shadow-retro transition-all font-bold text-ink border-2 border-ink/10"
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
            <div className="px-4 py-5 backdrop-blur-lg bg-white/95 border-b-4 border-ink/10 sticky top-0 z-20 shadow-retro">
                <div className="max-w-7xl mx-auto flex items-center justify-between">
                    <button
                        onClick={() => navigate(-1)}
                        className="inline-flex items-center gap-2 text-ink/60 hover:text-ink transition-colors group"
                    >
                        <svg className="w-5 h-5 group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
                        </svg>
                        <span className="text-sm font-bold tracking-wide">Leave</span>
                    </button>

                    <button
                        onClick={() => setShowRoomInfo(!showRoomInfo)}
                        className="flex-1 mx-4 flex items-center justify-center gap-2 hover:opacity-70 transition-opacity"
                    >
                        <h1 className="text-lg font-black truncate text-ink" style={{ fontFamily: 'Poppins' }}>{room?.name || 'Loading...'}</h1>
                        <svg className={`w-4 h-4 text-ink/50 transition-transform ${showRoomInfo ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                        </svg>
                    </button>

                    <div className="flex items-center gap-1">
                        {users && users.length > 0 && (
                            <div className="flex items-center gap-1.5 glass rounded-full px-3 py-1.5 border-2 border-ink/10">
                                <div className="w-2 h-2 bg-success rounded-full animate-pulse" />
                                <span className="text-xs font-medium">{users.length}</span>
                            </div>
                        )}
                    </div>
                </div>

                {/* Room info dropdown */}
                {showRoomInfo && (
                    <div className="mt-4 glass-elevated rounded-2xl p-5 animate-slide-down border-2 border-ink/10">
                        <div className="space-y-4">
                            <div>
                                <p className="text-xs text-ink/60 uppercase tracking-widest mb-2 font-bold">Room Code</p>
                                <div className="flex items-center gap-2">
                                    <code className="text-sm font-mono text-ink font-bold bg-surface px-4 py-2 rounded-xl border-2 border-ink/20">{id}</code>
                                    <button
                                        onClick={() => navigator.clipboard.writeText(id || '')}
                                        className="p-2 hover:bg-ink/5 rounded-lg transition-colors border-2 border-transparent hover:border-ink/10"
                                        title="Copy code"
                                    >
                                        <svg className="w-5 h-5 text-ink/60" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                            <path strokeLinecap="round" strokeLinejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                        </svg>
                                    </button>
                                </div>
                            </div>
                            {users && users.length > 0 && (
                                <div>
                                    <p className="text-xs text-ink/60 uppercase tracking-widest mb-2 font-bold">Listeners ({users.length})</p>
                                    <div className="flex flex-wrap gap-2">
                                        {users.slice(0, 8).map((user) => (
                                            <div key={user.id} className="text-xs bg-sakura/30 px-3 py-1.5 rounded-full text-ink/70 font-medium border border-sakura/50">
                                                {user.nickname || `User ${user.id.slice(0, 4)}`}
                                            </div>
                                        ))}
                                        {users.length > 8 && (
                                            <div className="text-xs bg-ink/10 px-3 py-1.5 rounded-full text-ink/50 font-medium">
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
                        {/* Video Player */}
                        <VideoPlayer onEnded={skip} isVisible={true} />

                        {/* Now Playing Card */}
                        <div className="glass-elevated rounded-3xl p-8 relative overflow-hidden animate-scale-in border-4 border-ink/10 shadow-retro-pink">
                            {/* Playing indicator glow */}
                            {isPlaying && (
                                <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-secondary/5 pointer-events-none" />
                            )}

                            <div className="relative">
                                <div className="flex items-center gap-2 mb-6">
                                    <div className={`w-3 h-3 rounded-full ${isPlaying ? 'bg-matcha animate-pulse shadow-neon-pink' : 'bg-ink/30'}`} />
                                    <span className="text-xs text-ink/60 uppercase tracking-widest font-bold">
                                        {isPlaying ? 'Now Playing' : 'Paused'}
                                    </span>
                                    <span className="text-xs jp-art text-ink/40">
                                        {isPlaying ? '再生中' : '一時停止'}
                                    </span>
                                </div>

                                {currentSong ? (
                                    <div className="text-center mb-6">
                                        <h2 className="text-3xl font-black mb-3 line-clamp-2 text-ink" style={{ fontFamily: 'Poppins' }}>
                                            {currentSong.title}
                                        </h2>
                                        <p className="text-lg text-ink/60 font-medium">
                                            {currentSong.artist || 'Unknown Artist'}
                                        </p>
                                    </div>
                                ) : (
                                    <div className="text-center mb-6 py-8">
                                        <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-white border-4 border-ink/20 shadow-retro mb-5">
                                            <svg className="w-10 h-10 text-ink/40" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                                <path strokeLinecap="round" strokeLinejoin="round" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
                                            </svg>
                                        </div>
                                        <h3 className="text-2xl font-black mb-2 text-ink" style={{ fontFamily: 'Poppins' }}>No song playing</h3>
                                        <p className="text-ink/60 mb-2 font-medium">Add a song to get started</p>
                                        <p className="text-sm jp-art text-ink/40">曲を追加してください</p>
                                    </div>
                                )}

                                {/* Player Controls */}
                                <PlayerControls roomId={id || ''} hasSongsInQueue={songs && songs.length > 0} />
                            </div>
                        </div>
                    </div>

                    {/* Queue Section */}
                    <div>
                        <div className="flex items-center justify-between mb-5">
                            <div>
                                <h2 className="text-2xl font-black text-ink" style={{ fontFamily: 'Poppins' }}>Up Next</h2>
                                <p className="text-sm text-ink/60 mt-1 font-medium">
                                    {songs.length} {songs.length === 1 ? 'song' : 'songs'} in queue
                                </p>
                                <p className="text-xs jp-art text-ink/40 mt-0.5">次の曲</p>
                            </div>
                            <button
                                onClick={() => setIsAddModalVisible(true)}
                                className="glass-elevated px-5 py-3 rounded-2xl hover:shadow-retro-pink transition-all hover:scale-105 active:scale-95 flex items-center gap-2 group border-2 border-ink/10"
                            >
                                <svg className="w-5 h-5 text-primary group-hover:rotate-90 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                                </svg>
                                <span className="font-black text-ink tracking-wide">Add Song</span>
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
