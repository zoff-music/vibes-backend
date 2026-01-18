import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useRoom } from '../hooks/useRoom';
import { usePlayback } from '../hooks/usePlayback';
import { useQueue } from '../hooks/useQueue';
import { VideoPlayer } from '../components/player/VideoPlayer';
import { PlayerControls } from '../components/player/PlayerControls';
import { AddToQueueModal } from '../components/queue/AddToQueueModal';
import { QueueList } from '../components/queue/QueueList';
import { Toast } from '../components/ui/Toast';
import { Song } from '@vibez/shared';
import { AnimatePresence, motion } from 'framer-motion';
import { QRCodeSVG } from 'qrcode.react';

export default function RoomView() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { currentSong, skip, isPlaying } = usePlayback(id || '');
    const { songs, fetchQueue, voteSong } = useQueue(id || '');
    const { room, fetchRoom, isLoading, error, joinRoom, userId, users, updateRoomSettings } = useRoom(id || '');

    const [isAddModalVisible, setIsAddModalVisible] = useState(false);
    const [showRoomInfo, setShowRoomInfo] = useState(false);
    const [showShare, setShowShare] = useState(false);
    const [showSettings, setShowSettings] = useState(false);
    const [toasts, setToasts] = useState<{ id: string; message: string; type: 'success' | 'info' }[]>([]);

    useEffect(() => {
        if (id) {
            fetchRoom();
            fetchQueue();
        }
    }, [id, fetchRoom, fetchQueue]);

    const formatTime = (ms: number) => {
        const seconds = Math.floor(ms / 1000);
        const m = Math.floor(seconds / 60);
        const s = seconds % 60;
        return `${m}:${s.toString().padStart(2, '0')}`;
    };

    const { actualPositionMs } = usePlayback(id || '');
    const progress = currentSong ? actualPositionMs / (currentSong.duration * 1000) : 0;

    useEffect(() => {
        const handleSongAdded = (e: any) => {
            console.log('[UI] song-added event received:', e.detail);
            const song = e.detail as Song;
            setToasts(prev => [...prev, {
                id: Math.random().toString(36).substr(2, 9),
                message: `"${song.title}" added to queue`,
                type: 'success'
            }]);
        };

        window.addEventListener('song-added', handleSongAdded);
        return () => window.removeEventListener('song-added', handleSongAdded);
    }, []);

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

                    <div className="flex items-center gap-2">
                        {users && users.length > 0 && (
                            <div className="flex items-center gap-1.5 glass rounded-full px-3 py-1.5 border-2 border-ink/10 mr-2">
                                <div className="w-2 h-2 bg-success rounded-full animate-pulse" />
                                <span className="text-xs font-medium">{users.length}</span>
                            </div>
                        )}

                        <div className="relative">
                            <button
                                onClick={() => setShowShare(!showShare)}
                                className={`p-2.5 rounded-xl transition-all border-2 ${showShare ? 'bg-ink text-white border-ink' : 'text-ink/60 hover:text-ink border-ink/10 hover:border-ink/20'}`}
                                title="Share Room"
                            >
                                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
                                </svg>
                            </button>

                            <AnimatePresence>
                                {showShare && (
                                    <motion.div
                                        initial={{ opacity: 0, y: 10, scale: 0.95 }}
                                        animate={{ opacity: 1, y: 0, scale: 1 }}
                                        exit={{ opacity: 0, y: 10, scale: 0.95 }}
                                        className="absolute right-0 mt-3 p-4 bg-white rounded-3xl shadow-2xl border-4 border-ink z-50 w-72"
                                    >
                                        <div className="text-center space-y-4">
                                            <div className="p-4 bg-sakura/20 rounded-2xl inline-block ring-2 ring-ink/5">
                                                <QRCodeSVG
                                                    value={window.location.href}
                                                    size={180}
                                                    bgColor="#fff0f2"
                                                    fgColor="#2d3142"
                                                    level="H"
                                                />
                                            </div>
                                            <div>
                                                <p className="font-black text-sm text-ink mb-1">Invite Friends</p>
                                                <div className="flex items-center gap-2 bg-ink/5 p-2 rounded-xl">
                                                    <p className="text-[10px] text-ink/60 truncate font-mono flex-1 text-left">{window.location.href}</p>
                                                    <button
                                                        onClick={() => {
                                                            navigator.clipboard.writeText(window.location.href);
                                                            setToasts(prev => [...prev, {
                                                                id: Math.random().toString(36).substr(2, 9),
                                                                message: 'Link copied!',
                                                                type: 'success'
                                                            }]);
                                                            setShowShare(false);
                                                        }}
                                                        className="p-1 px-2 bg-ink text-white rounded-lg text-[10px] font-bold hover:scale-105 active:scale-95 transition-all"
                                                    >
                                                        Copy
                                                    </button>
                                                </div>
                                            </div>
                                        </div>
                                    </motion.div>
                                )}
                            </AnimatePresence>
                        </div>

                        <div className="relative ml-1">
                            <button
                                onClick={() => setShowSettings(!showSettings)}
                                className={`p-2.5 rounded-xl transition-all border-2 ${showSettings ? 'bg-ink text-white border-ink' : 'text-ink/60 hover:text-ink border-ink/10 hover:border-ink/20'}`}
                                title="Room Settings"
                            >
                                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                </svg>
                            </button>

                            <AnimatePresence>
                                {showSettings && (
                                    <motion.div
                                        initial={{ opacity: 0, y: 10, scale: 0.95 }}
                                        animate={{ opacity: 1, y: 0, scale: 1 }}
                                        exit={{ opacity: 0, y: 10, scale: 0.95 }}
                                        className="absolute right-0 mt-3 p-5 bg-white rounded-3xl shadow-2xl border-4 border-ink z-50 w-72"
                                    >
                                        <div className="space-y-4">
                                            <h4 className="font-black text-ink border-b-2 border-ink/5 pb-2 text-sm uppercase tracking-wider">Room Control</h4>

                                            <div className="flex items-center justify-between group">
                                                <div className="flex flex-col">
                                                    <span className="text-sm font-bold text-ink">Allow Skip</span>
                                                    <span className="text-[10px] text-ink/40">Anyone can skip</span>
                                                </div>
                                                <button
                                                    onClick={() => room && updateRoomSettings({ ...room.settings, skipAllowed: !room.settings.skipAllowed })}
                                                    className={`w-12 h-6 rounded-full relative transition-colors border-2 border-ink ${room?.settings.skipAllowed ? 'bg-primary' : 'bg-ink/5 opacity-50'}`}
                                                >
                                                    <div className={`absolute top-1 w-2 h-2 rounded-full bg-white shadow-sm transition-all ${room?.settings.skipAllowed ? 'right-1.5' : 'left-1.5'}`} />
                                                </button>
                                            </div>

                                            <div className="flex items-center justify-between group">
                                                <div className="flex flex-col">
                                                    <span className="text-sm font-bold text-ink">Democratic Skip</span>
                                                    <span className="text-[10px] text-ink/40">Require votes</span>
                                                </div>
                                                <button
                                                    onClick={() => room && updateRoomSettings({ ...room.settings, democraticSkip: !room.settings.democraticSkip })}
                                                    className={`w-12 h-6 rounded-full relative transition-colors border-2 border-ink ${room?.settings.democraticSkip ? 'bg-primary' : 'bg-ink/5 opacity-50'}`}
                                                >
                                                    <div className={`absolute top-1 w-2 h-2 rounded-full bg-white shadow-sm transition-all ${room?.settings.democraticSkip ? 'right-1.5' : 'left-1.5'}`} />
                                                </button>
                                            </div>

                                            <div className="flex items-center justify-between group">
                                                <div className="flex flex-col">
                                                    <span className="text-sm font-bold text-ink">Loop Queue</span>
                                                    <span className="text-[10px] text-ink/40">Cycled back to end</span>
                                                </div>
                                                <button
                                                    onClick={() => room && updateRoomSettings({ ...room.settings, loopQueue: !room.settings.loopQueue })}
                                                    className={`w-12 h-6 rounded-full relative transition-colors border-2 border-ink ${room?.settings.loopQueue ? 'bg-primary' : 'bg-ink/5 opacity-50'}`}
                                                >
                                                    <div className={`absolute top-1 w-2 h-2 rounded-full bg-white shadow-sm transition-all ${room?.settings.loopQueue ? 'right-1.5' : 'left-1.5'}`} />
                                                </button>
                                            </div>

                                            <div className="flex items-center justify-between group">
                                                <div className="flex flex-col">
                                                    <span className="text-sm font-bold text-ink">Remove Played</span>
                                                    <span className="text-[10px] text-ink/40">Removed after play</span>
                                                </div>
                                                <button
                                                    onClick={() => room && updateRoomSettings({ ...room.settings, removeOnPlay: !room.settings.removeOnPlay })}
                                                    className={`w-12 h-6 rounded-full relative transition-colors border-2 border-ink ${room?.settings.removeOnPlay ? 'bg-ink' : 'bg-ink/5 opacity-50'}`}
                                                >
                                                    <div className={`absolute top-1 w-2 h-2 rounded-full bg-white shadow-sm transition-all ${room?.settings.removeOnPlay ? 'right-1.5' : 'left-1.5'}`} />
                                                </button>
                                            </div>

                                            <p className="text-[10px] text-ink/30 text-center pt-2 font-black italic">Settings sync enabled</p>
                                        </div>
                                    </motion.div>
                                )}
                            </AnimatePresence>
                        </div>
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
                <div className="max-w-7xl mx-auto px-4 py-8 lg:grid lg:grid-cols-[1fr,400px] lg:gap-12 items-start">
                    {/* Player Section */}
                    <div className="space-y-6">
                        {/* Video Player */}
                        <VideoPlayer onEnded={skip} isVisible={true} />

                        {/* Controls (always below video) */}
                        <PlayerControls
                            roomId={id || ''}
                            hasSongsInQueue={songs && songs.length > 0}
                            onAddSong={() => setIsAddModalVisible(true)}
                        />
                    </div>

                    {/* Queue & Now Playing Section */}
                    <div className="mt-8 lg:mt-0 space-y-8">
                        <div>
                            {/* Now Playing (Integrated into list style) */}
                            <AnimatePresence mode="wait">
                                {currentSong && (
                                    <motion.div
                                        key={currentSong.id}
                                        initial={{ opacity: 0, scale: 0.95, y: 10 }}
                                        animate={{ opacity: 1, scale: 1, y: 0 }}
                                        exit={{ opacity: 0, scale: 1.05, y: -10 }}
                                        transition={{ duration: 0.3 }}
                                        className="mb-8"
                                    >
                                        <div className="flex items-center gap-2 mb-3">
                                            <div className={`w-2 h-2 rounded-full ${isPlaying ? 'bg-matcha animate-pulse shadow-neon-pink' : 'bg-ink/30'}`} />
                                            <span className="text-[10px] text-ink/60 uppercase tracking-[0.2em] font-black">
                                                {isPlaying ? 'Now Playing' : 'Paused'}
                                            </span>
                                        </div>

                                        <div className="relative glass-elevated rounded-2xl p-4 border-2 border-primary/20 overflow-hidden flex gap-4 items-center shadow-retro-pink/20 bg-white/40 backdrop-blur-sm group/card">
                                            {/* Progress Background */}
                                            <motion.div
                                                className="absolute inset-y-0 left-0 bg-primary/10 pointer-events-none"
                                                initial={{ width: 0 }}
                                                animate={{ width: `${progress * 100}%` }}
                                                transition={{ duration: 0.1, ease: 'linear' }}
                                            />

                                            {currentSong.thumbnailUrl && (
                                                <div className="relative z-10 flex-shrink-0">
                                                    <img
                                                        src={currentSong.thumbnailUrl}
                                                        alt=""
                                                        className="w-16 h-16 rounded-xl object-cover border-2 border-ink/10 shadow-sm transition-transform group-hover/card:scale-105"
                                                    />
                                                </div>
                                            )}
                                            <div className="relative z-10 flex-1 min-w-0">
                                                <h3 className="font-bold text-ink truncate text-sm">{currentSong.title}</h3>
                                                <p className="text-xs text-ink/60 truncate font-medium">
                                                    {currentSong.artist || 'Unknown Artist'} • {formatTime(currentSong.duration * 1000)}
                                                </p>
                                            </div>
                                        </div>

                                        <div className="h-px bg-ink/10 mt-8 mb-4" />
                                    </motion.div>
                                )}
                            </AnimatePresence>

                            {/* Up Next List */}
                            <div>
                                <h3 className="text-[10px] text-ink/60 uppercase tracking-[0.2em] font-black mb-4">
                                    Up Next ({songs.filter(s => s.position > 0 && s.id !== currentSong?.id).length})
                                </h3>
                                <QueueList 
                                    songs={songs.filter(s => s.id !== currentSong?.id)} 
                                    roomId={id || ''} 
                                    onVote={async (songId) => {
                                        const result = await voteSong(songId);
                                        if (result === 'success') {
                                            setToasts(prev => [...prev, {
                                                id: Math.random().toString(36).substr(2, 9),
                                                message: 'Vote recorded!',
                                                type: 'success'
                                            }]);
                                        } else if (result === 'already_voted') {
                                            setToasts(prev => [...prev, {
                                                id: Math.random().toString(36).substr(2, 9),
                                                message: 'You have already voted for this song',
                                                type: 'info'
                                            }]);
                                        }
                                    }}
                                />
                                {songs.filter(s => s.position > 0 && s.id !== currentSong?.id).length === 0 && !currentSong && (
                                    <div className="text-center py-12 glass rounded-2xl border-2 border-dashed border-ink/10">
                                        <p className="text-ink/40 font-bold">Queue is empty</p>
                                        <p className="text-[10px] jp-art text-ink/20">キューは空です</p>
                                    </div>
                                )}
                            </div>
                        </div>

                    </div>
                </div>
            </div>

            {/* Toasts */}
            {toasts.map(toast => (
                <Toast
                    key={toast.id}
                    message={toast.message}
                    type={toast.type === 'success' ? 'success' : 'info'}
                    onClose={() => setToasts(prev => prev.filter(t => t.id !== toast.id))}
                />
            ))}

            {/* Add Song Modal */}
            <AddToQueueModal
                roomId={id || ''}
                isVisible={isAddModalVisible}
                onClose={() => setIsAddModalVisible(false)}
            />
        </div>
    );
}
