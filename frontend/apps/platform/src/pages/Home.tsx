import { useState } from 'react';
import { useNavigate } from 'react-router';

export default function Home() {
  const [roomCode, setRoomCode] = useState('');
  const navigate = useNavigate();

  const handleCreateRoom = () => {
    navigate('/rooms/create');
  };

  const handleJoinRoom = () => {
    if (roomCode.trim()) {
      const slug = roomCode.trim().toLowerCase().replace(/\s+/g, '-');
      navigate(`/rooms/${slug}`);
    }
  };

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-12">
      {/* Floating retro shapes for playful synthwave vibe */}
      <div
        className="absolute top-20 left-[10%] h-32 w-32 rotate-12 animate-float rounded-lg bg-sakura/30 dark:bg-sakura/20"
        style={{ animationDelay: '0s' }}
      />
      <div
        className="absolute top-40 right-[15%] h-24 w-24 animate-float rounded-full bg-secondary/20 dark:bg-secondary/15"
        style={{ animationDelay: '1s' }}
      />
      <div
        className="absolute bottom-32 left-[20%] h-20 w-20 rotate-45 animate-float bg-accent/20 dark:bg-accent/15"
        style={{ animationDelay: '2s' }}
      />

      <div className="relative z-10 w-full max-w-md">
        {/* Logo section with Japanese text art */}
        <div className="mb-12 text-center relative">
          
          {/* Large background Japanese text */}
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none -z-10">
            <span className="jp-art text-ink/40 dark:text-white/20 font-black select-none" style={{ fontSize: '10rem', lineHeight: '1', transform: 'translateY(-40px)' }}>
              ノリ
            </span>
          </div>
          
          {/* Main title on top */}
          <div className="relative z-10">
            <h1
              className="mb-2 font-black text-5xl text-ink dark:text-white tracking-tight"
              style={{ fontFamily: 'Poppins' }}
            >
              nori
            </h1>
          </div>
          <p
            className="font-medium text-base text-ink/70 dark:text-gray-300"
          >
            Shared music, synchronized vibes
          </p>
          <p className="jp-art text-ink/30 dark:text-gray-500 text-xs">音楽を共有</p>
        </div>

        {/* Main actions */}
        <div className="space-y-5">
          {/* Create room button - primary action */}
          <button
            onClick={handleCreateRoom}
            className="group glass-elevated w-full cursor-pointer rounded-2xl p-6 transition-all hover:scale-[1.02] hover:shadow-retro-pink active:scale-[0.98]"
          >
            <div className="flex items-center justify-between">
              <div className="text-left">
                <div
                  className="mb-1.5 font-bold text-ink dark:text-white text-xl"
                  style={{ fontFamily: 'Poppins' }}
                >
                  Start a Session
                </div>
                <div className="font-medium text-ink/60 dark:text-gray-400 text-sm">
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
            <div className="h-0.5 flex-1 bg-ink/10 dark:bg-gray-700" />
            <span className="jp-art font-medium text-ink/50 dark:text-gray-500 text-xs tracking-wider">
              または
            </span>
            <div className="h-0.5 flex-1 bg-ink/10 dark:bg-gray-700" />
          </div>

          {/* Join room card */}
          <div className="glass rounded-2xl border-2 border-ink/10 dark:border-gray-700 p-6">
            <div className="mb-5">
              <label className="mb-3 block font-bold text-ink/80 dark:text-gray-300 text-sm tracking-wide">
                Have a room code?
              </label>
              <div className="relative">
                <input
                  type="text"
                  placeholder="Enter code"
                  value={roomCode}
                  onChange={(e) => setRoomCode(e.target.value.toLowerCase())}
                  onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
                  className="w-full rounded-xl border-2 border-ink/20 dark:border-gray-600 bg-surface dark:bg-gray-700 px-4 py-4 font-mono text-base text-ink dark:text-white tracking-wider transition-all placeholder:text-ink/40 dark:placeholder:text-gray-500 focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] focus:outline-hidden"
                  maxLength={20}
                />
                {roomCode && (
                  <button
                    onClick={() => setRoomCode('')}
                    className="absolute top-1/2 right-3 -translate-y-1/2 cursor-pointer rounded-lg p-1 text-ink/40 dark:text-gray-500 transition-colors hover:bg-ink/5 dark:hover:bg-gray-600 hover:text-ink dark:hover:text-gray-300"
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
              className="w-full cursor-pointer rounded-xl bg-secondary dark:bg-primary py-4 font-bold text-white dark:text-white tracking-wide transition-all hover:bg-secondary/90 dark:hover:bg-primary/90 hover:shadow-retro-cyan active:scale-[0.98] disabled:cursor-not-allowed disabled:bg-ink/10 dark:disabled:bg-gray-700 disabled:text-ink/30 dark:disabled:text-gray-500"
              style={{ fontFamily: 'Poppins' }}
            >
              Join Session
            </button>
          </div>
        </div>

        {/* Footer hint with Japanese */}
        <div className="mt-10 text-center">
          <p className="mb-1 font-medium text-ink/50 dark:text-gray-400 text-xs">
            Listen together in real-time
          </p>
          <p className="jp-art text-ink/30 dark:text-gray-600 text-xs">リアルタイム同期</p>
        </div>
      </div>
    </div>
  );
}
