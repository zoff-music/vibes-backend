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
            {/* Ambient orbs for visual interest */}
            <div className="absolute top-20 left-1/4 w-96 h-96 bg-primary/10 rounded-full blur-3xl animate-pulse" style={{ animationDuration: '4s' }} />
            <div className="absolute bottom-20 right-1/4 w-80 h-80 bg-secondary/8 rounded-full blur-3xl animate-pulse" style={{ animationDuration: '5s', animationDelay: '1s' }} />

            <div className="w-full max-w-md relative z-10 animate-fade-in">
                {/* Logo section */}
                <div className="text-center mb-16">
                    <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl glass-elevated mb-6 animate-scale-in">
                        <svg className="w-10 h-10 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
                        </svg>
                    </div>
                    <h1 className="text-4xl font-bold mb-3 bg-gradient-to-r from-white via-white to-white/60 bg-clip-text text-transparent animate-slide-down">
                        Vibez
                    </h1>
                    <p className="text-base text-text-muted animate-slide-down" style={{ animationDelay: '0.1s' }}>
                        Shared music, synchronized vibes
                    </p>
                </div>

                {/* Main actions */}
                <div className="space-y-4 animate-slide-up" style={{ animationDelay: '0.2s' }}>
                    {/* Create room button - primary action */}
                    <button
                        onClick={handleCreateRoom}
                        className="group w-full relative overflow-hidden glass-elevated rounded-xl p-6 transition-smooth hover:scale-[1.02] active:scale-[0.98]"
                    >
                        <div className="absolute inset-0 bg-gradient-to-r from-primary/20 to-secondary/20 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                        <div className="relative flex items-center justify-between">
                            <div className="text-left">
                                <div className="text-lg font-semibold text-white mb-1">Start a Session</div>
                                <div className="text-sm text-text-muted">Create your music room</div>
                            </div>
                            <div className="flex items-center justify-center w-12 h-12 rounded-xl bg-primary/10 group-hover:bg-primary/20 transition-colors">
                                <svg className="w-6 h-6 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                                </svg>
                            </div>
                        </div>
                    </button>

                    {/* Divider */}
                    <div className="flex items-center gap-4 py-2">
                        <div className="flex-1 h-px bg-border" />
                        <span className="text-xs text-text-subtle uppercase tracking-wider">Or</span>
                        <div className="flex-1 h-px bg-border" />
                    </div>

                    {/* Join room card */}
                    <div className="glass rounded-xl p-6">
                        <div className="mb-4">
                            <label className="text-sm font-medium text-text-muted mb-2 block">
                                Have a room code?
                            </label>
                            <div className="relative">
                                <input
                                    type="text"
                                    placeholder="Enter code"
                                    value={roomCode}
                                    onChange={(e) => setRoomCode(e.target.value.toUpperCase())}
                                    onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
                                    className="w-full bg-surfaceElevated/50 backdrop-blur-sm rounded-lg px-4 py-3.5 text-base text-white placeholder:text-text-subtle border border-border focus:outline-none focus:border-primary/50 focus:ring-2 focus:ring-primary/20 transition-all"
                                    maxLength={20}
                                />
                                {roomCode && (
                                    <button
                                        onClick={() => setRoomCode('')}
                                        className="absolute right-3 top-1/2 -translate-y-1/2 text-text-subtle hover:text-white transition-colors"
                                    >
                                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                        </svg>
                                    </button>
                                )}
                            </div>
                        </div>
                        <button
                            onClick={handleJoinRoom}
                            disabled={!roomCode.trim()}
                            className="w-full bg-primary hover:bg-primary-muted disabled:bg-surface disabled:text-text-subtle text-white py-3.5 rounded-lg font-medium transition-all disabled:cursor-not-allowed hover:shadow-glow active:scale-[0.98]"
                        >
                            Join Session
                        </button>
                    </div>
                </div>

                {/* Footer hint */}
                <div className="text-center mt-12 animate-fade-in" style={{ animationDelay: '0.4s' }}>
                    <p className="text-xs text-text-subtle">
                        Listen together in real-time
                    </p>
                </div>
            </div>
        </div>
    );
}
