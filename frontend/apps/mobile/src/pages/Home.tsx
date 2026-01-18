import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

export default function Home() {
    const [roomCode, setRoomCode] = useState('');
    const navigate = useNavigate();

    const handleCreateRoom = () => {
        navigate('/room/create');
    };

    const handleJoinRoom = () => {
        if (roomCode.trim()) {
            navigate(`/room/${roomCode.trim()}`);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center px-4 py-12 relative overflow-hidden">
            {/* Floating retro shapes for playful synthwave vibe */}
            <div className="absolute top-20 left-[10%] w-32 h-32 bg-sakura/30 rounded-lg rotate-12 animate-float" style={{ animationDelay: '0s' }} />
            <div className="absolute top-40 right-[15%] w-24 h-24 bg-secondary/20 rounded-full animate-float" style={{ animationDelay: '1s' }} />
            <div className="absolute bottom-32 left-[20%] w-20 h-20 bg-accent/20 rotate-45 animate-float" style={{ animationDelay: '2s' }} />

            <div className="w-full max-w-md relative z-10 animate-fade-in">
                {/* Logo section with Japanese text art */}
                <div className="text-center mb-12">
                    <div className="inline-flex items-center justify-center w-24 h-24 rounded-2xl bg-white border-4 border-ink shadow-retro-pink mb-6 animate-scale-in hover:shadow-neon-pink transition-all">
                        <svg className="w-12 h-12 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
                        </svg>
                    </div>
                    <h1 className="text-5xl font-black mb-2 text-ink tracking-tight animate-slide-down" style={{ fontFamily: 'Poppins' }}>
                        Vibez
                    </h1>
                    <p className="text-sm jp-art text-ink/60 mb-3 animate-slide-down" style={{ animationDelay: '0.05s' }}>
                        音楽を共有
                    </p>
                    <p className="text-base text-ink/70 font-medium animate-slide-down" style={{ animationDelay: '0.1s' }}>
                        Shared music, synchronized vibes
                    </p>
                </div>

                {/* Main actions */}
                <div className="space-y-5 animate-slide-up" style={{ animationDelay: '0.2s' }}>
                    {/* Create room button - primary action */}
                    <button
                        onClick={handleCreateRoom}
                        className="group w-full glass-elevated rounded-2xl p-6 transition-all hover:scale-[1.02] hover:shadow-retro-pink active:scale-[0.98]"
                    >
                        <div className="flex items-center justify-between">
                            <div className="text-left">
                                <div className="text-xl font-bold text-ink mb-1.5" style={{ fontFamily: 'Poppins' }}>
                                    Start a Session
                                </div>
                                <div className="text-sm text-ink/60 font-medium">Create your music room</div>
                            </div>
                            <div className="flex items-center justify-center w-14 h-14 rounded-xl bg-primary text-white shadow-retro group-hover:shadow-neon-pink transition-all">
                                <svg className="w-7 h-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                                </svg>
                            </div>
                        </div>
                    </button>

                    {/* Divider with Japanese accent */}
                    <div className="flex items-center gap-4 py-2">
                        <div className="flex-1 h-0.5 bg-ink/10" />
                        <span className="text-xs jp-art text-ink/50 font-medium tracking-wider">または</span>
                        <div className="flex-1 h-0.5 bg-ink/10" />
                    </div>

                    {/* Join room card */}
                    <div className="glass rounded-2xl p-6 border-2 border-ink/10">
                        <div className="mb-5">
                            <label className="text-sm font-bold text-ink/80 mb-3 block tracking-wide">
                                Have a room code?
                            </label>
                            <div className="relative">
                                <input
                                    type="text"
                                    placeholder="Enter code"
                                    value={roomCode}
                                    onChange={(e) => setRoomCode(e.target.value.toUpperCase())}
                                    onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
                                    className="w-full bg-surface rounded-xl px-4 py-4 text-base text-ink placeholder:text-ink/40 border-2 border-ink/20 focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] transition-all font-mono tracking-wider"
                                    maxLength={20}
                                />
                                {roomCode && (
                                    <button
                                        onClick={() => setRoomCode('')}
                                        className="absolute right-3 top-1/2 -translate-y-1/2 text-ink/40 hover:text-ink transition-colors p-1 rounded-lg hover:bg-ink/5"
                                    >
                                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                                        </svg>
                                    </button>
                                )}
                            </div>
                        </div>
                        <button
                            onClick={handleJoinRoom}
                            disabled={!roomCode.trim()}
                            className="w-full bg-secondary hover:bg-secondary/90 disabled:bg-ink/10 disabled:text-ink/30 text-white py-4 rounded-xl font-bold transition-all disabled:cursor-not-allowed hover:shadow-retro-cyan active:scale-[0.98] tracking-wide"
                            style={{ fontFamily: 'Poppins' }}
                        >
                            Join Session
                        </button>
                    </div>
                </div>

                {/* Footer hint with Japanese */}
                <div className="text-center mt-10 animate-fade-in" style={{ animationDelay: '0.4s' }}>
                    <p className="text-xs text-ink/50 font-medium mb-1">
                        Listen together in real-time
                    </p>
                    <p className="text-xs jp-art text-ink/30">
                        リアルタイム同期
                    </p>
                </div>
            </div>
        </div>
    );
}
