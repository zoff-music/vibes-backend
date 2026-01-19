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
      const slug = roomCode.trim().toLowerCase().replace(/\s+/g, '-');
      navigate(`/room/${slug}`);
    }
  };

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-12">
      {/* Floating retro shapes for playful synthwave vibe */}
      <div
        className="absolute top-20 left-[10%] h-32 w-32 rotate-12 animate-float rounded-lg bg-sakura/30"
        style={{ animationDelay: '0s' }}
      />
      <div
        className="absolute top-40 right-[15%] h-24 w-24 animate-float rounded-full bg-secondary/20"
        style={{ animationDelay: '1s' }}
      />
      <div
        className="absolute bottom-32 left-[20%] h-20 w-20 rotate-45 animate-float bg-accent/20"
        style={{ animationDelay: '2s' }}
      />

      <div className="relative z-10 w-full max-w-md animate-fade-in">
        {/* Logo section with Japanese text art */}
        <div className="mb-12 text-center">
          <div className="mb-6 inline-flex h-24 w-24 animate-scale-in items-center justify-center rounded-2xl border-4 border-ink bg-white shadow-retro-pink transition-all hover:shadow-neon-pink">
            <svg
              className="h-12 w-12 text-primary"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3"
              />
            </svg>
          </div>
          <h1
            className="mb-2 animate-slide-down font-black text-5xl text-ink tracking-tight"
            style={{ fontFamily: 'Poppins' }}
          >
            Vibez
          </h1>
          <p
            className="jp-art mb-3 animate-slide-down text-ink/60 text-sm"
            style={{ animationDelay: '0.05s' }}
          >
            音楽を共有
          </p>
          <p
            className="animate-slide-down font-medium text-base text-ink/70"
            style={{ animationDelay: '0.1s' }}
          >
            Shared music, synchronized vibes
          </p>
        </div>

        {/* Main actions */}
        <div
          className="animate-slide-up space-y-5"
          style={{ animationDelay: '0.2s' }}
        >
          {/* Create room button - primary action */}
          <button
            onClick={handleCreateRoom}
            className="group glass-elevated w-full rounded-2xl p-6 transition-all hover:scale-[1.02] hover:shadow-retro-pink active:scale-[0.98]"
          >
            <div className="flex items-center justify-between">
              <div className="text-left">
                <div
                  className="mb-1.5 font-bold text-ink text-xl"
                  style={{ fontFamily: 'Poppins' }}
                >
                  Start a Session
                </div>
                <div className="font-medium text-ink/60 text-sm">
                  Create your music room
                </div>
              </div>
              <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-primary text-white shadow-retro transition-all group-hover:shadow-neon-pink">
                <svg
                  className="h-7 w-7"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2.5}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M12 4v16m8-8H4"
                  />
                </svg>
              </div>
            </div>
          </button>

          {/* Divider with Japanese accent */}
          <div className="flex items-center gap-4 py-2">
            <div className="h-0.5 flex-1 bg-ink/10" />
            <span className="jp-art font-medium text-ink/50 text-xs tracking-wider">
              または
            </span>
            <div className="h-0.5 flex-1 bg-ink/10" />
          </div>

          {/* Join room card */}
          <div className="glass rounded-2xl border-2 border-ink/10 p-6">
            <div className="mb-5">
              <label className="mb-3 block font-bold text-ink/80 text-sm tracking-wide">
                Have a room code?
              </label>
              <div className="relative">
                <input
                  type="text"
                  placeholder="Enter code"
                  value={roomCode}
                  onChange={(e) => setRoomCode(e.target.value.toLowerCase())}
                  onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
                  className="w-full rounded-xl border-2 border-ink/20 bg-surface px-4 py-4 font-mono text-base text-ink tracking-wider transition-all placeholder:text-ink/40 focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] focus:outline-hidden"
                  maxLength={20}
                />
                {roomCode && (
                  <button
                    onClick={() => setRoomCode('')}
                    className="absolute top-1/2 right-3 -translate-y-1/2 rounded-lg p-1 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
                  >
                    <svg
                      className="h-5 w-5"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      strokeWidth={2}
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                )}
              </div>
            </div>
            <button
              onClick={handleJoinRoom}
              disabled={!roomCode.trim()}
              className="w-full rounded-xl bg-secondary py-4 font-bold text-white tracking-wide transition-all hover:bg-secondary/90 hover:shadow-retro-cyan active:scale-[0.98] disabled:cursor-not-allowed disabled:bg-ink/10 disabled:text-ink/30"
              style={{ fontFamily: 'Poppins' }}
            >
              Join Session
            </button>
          </div>
        </div>

        {/* Footer hint with Japanese */}
        <div
          className="mt-10 animate-fade-in text-center"
          style={{ animationDelay: '0.4s' }}
        >
          <p className="mb-1 font-medium text-ink/50 text-xs">
            Listen together in real-time
          </p>
          <p className="jp-art text-ink/30 text-xs">リアルタイム同期</p>
        </div>
      </div>
    </div>
  );
}
