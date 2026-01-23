import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { api } from '@vibez/api';

export default function Home() {
  const [roomCode, setRoomCode] = useState('');
  const [isValidating, setIsValidating] = useState(false);
  const navigate = useNavigate();

  const handleJoinRoom = async () => {
    if (!roomCode.trim() || isValidating) return;

    const slug = roomCode.trim().toLowerCase().replace(/\s+/g, '-');
    setIsValidating(true);

    // Check if room exists before navigating
    const [err, room] = await api.get('/rooms/{id}', { id: slug });
    
    setIsValidating(false);

    if (err) {
      // Room doesn't exist, redirect to create room page with the name pre-filled
      navigate(`/rooms/create?name=${encodeURIComponent(roomCode.trim())}`);
    } else {
      // Room exists, navigate to it
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
        <div className="relative mb-12 text-center">
          {/* Large background Japanese text */}
          <div className="pointer-events-none absolute inset-0 -z-10 flex items-center justify-center">
            <span
              className="jp-art select-none font-black text-ink/40 dark:text-white/20"
              style={{
                fontSize: '10rem',
                lineHeight: '1',
                transform: 'translateY(-40px)',
              }}
            >
              ノリ
            </span>
          </div>

          {/* Main title on top */}
          <div className="relative z-10">
            <h1
              className="mb-2 font-black text-5xl text-ink tracking-tight dark:text-white"
              style={{ fontFamily: 'Poppins' }}
            >
              nori
            </h1>
          </div>
          <p className="font-medium text-base text-ink/70 dark:text-gray-300">
            Shared music, synchronized vibes
          </p>
          <p className="jp-art text-ink/30 text-xs dark:text-gray-500">
            音楽を共有
          </p>
        </div>

        {/* Main actions */}
        <div className="space-y-5">
          {/* Create room button - primary action */}
          <Link
            to="/rooms/create"
            className="group glass-elevated block w-full cursor-pointer rounded-2xl p-6 transition-all hover:scale-[1.02] hover:shadow-retro-pink active:scale-[0.98]"
          >
            <div className="flex items-center justify-between">
              <div className="text-left">
                <div
                  className="mb-1.5 font-bold text-ink text-xl dark:text-white"
                  style={{ fontFamily: 'Poppins' }}
                >
                  Start a Session
                </div>
                <div className="font-medium text-ink/60 text-sm dark:text-gray-400">
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
          </Link>

          {/* Divider with Japanese accent */}
          <div className="flex items-center gap-4 py-2">
            <div className="h-0.5 flex-1 bg-ink/10 dark:bg-gray-700" />
            <span className="jp-art font-medium text-ink/50 text-xs tracking-wider dark:text-gray-500">
              または
            </span>
            <div className="h-0.5 flex-1 bg-ink/10 dark:bg-gray-700" />
          </div>

          {/* Join room card */}
          <div className="glass rounded-2xl border-2 border-ink/10 p-6 dark:border-gray-700">
            <div className="mb-5">
              <label className="mb-3 block font-bold text-ink/80 text-sm tracking-wide dark:text-gray-300">
                Have a room code?
              </label>
              <div className="relative">
                <input
                  type="text"
                  placeholder="Enter code"
                  value={roomCode}
                  onChange={(e) => setRoomCode(e.target.value.toLowerCase())}
                  onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
                  disabled={isValidating}
                  className="w-full rounded-xl border-2 border-ink/20 bg-surface px-4 py-4 font-mono text-base text-ink tracking-wider transition-all placeholder:text-ink/40 focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] focus:outline-hidden disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder:text-gray-500"
                  maxLength={20}
                />
                {roomCode && !isValidating && (
                  <button
                    onClick={() => setRoomCode('')}
                    className="absolute top-1/2 right-3 -translate-y-1/2 cursor-pointer rounded-lg p-1 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink dark:text-gray-500 dark:hover:bg-gray-600 dark:hover:text-gray-300"
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
              disabled={!roomCode.trim() || isValidating}
              className="w-full cursor-pointer rounded-xl bg-secondary py-4 font-bold text-white tracking-wide transition-all hover:bg-secondary/90 hover:shadow-retro-cyan active:scale-[0.98] disabled:cursor-not-allowed disabled:bg-ink/10 disabled:text-ink/30 dark:bg-primary dark:text-white dark:disabled:bg-gray-700 dark:disabled:text-gray-500 dark:hover:bg-primary/90"
              style={{ fontFamily: 'Poppins' }}
            >
              {isValidating ? (
                <div className="flex items-center justify-center gap-2">
                  <div className="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                  <span>Checking...</span>
                </div>
              ) : (
                'Join Session'
              )}
            </button>
          </div>
        </div>

        {/* Footer hint with Japanese */}
        <div className="mt-10 text-center">
          <p className="mb-1 font-medium text-ink/50 text-xs dark:text-gray-400">
            Listen together in real-time
          </p>
          <p className="jp-art text-ink/30 text-xs dark:text-gray-600">
            リアルタイム同期
          </p>
        </div>
      </div>
    </div>
  );
}
