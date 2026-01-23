import { api } from '@vibez/api';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router';

export default function Home() {
  const [roomCode, setRoomCode] = useState('');
  const [isValidating, setIsValidating] = useState(false);
  const navigate = useNavigate();

  const handleJoinRoom = async () => {
    if (!roomCode.trim() || isValidating) return;

    const slug = roomCode.trim().toLowerCase().replace(/\s+/g, '-');
    setIsValidating(true);

    // Check if room exists before navigating
    const [err] = await api.get('/rooms/{id}', { id: slug });

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
    <div className="relative flex min-h-screen flex-col overflow-x-hidden bg-theme text-theme md:block">
      <div className="synth-sky pointer-events-none fixed inset-0" />
      <div className="synth-haze pointer-events-none fixed inset-0" />
      <div className="vhs-scanlines pointer-events-none fixed inset-0" />
      <div className="sun-hero sunset-orb pointer-events-none fixed" />
      <div className="retro-grid pointer-events-none fixed bottom-0 left-1/2 z-10 h-[45vh] w-[140%]" />

      <div className="relative z-10 mx-auto flex h-full w-full max-w-5xl flex-1 flex-col items-center justify-center px-6 py-6">
        <div className="crt-frame w-full max-w-3xl rounded-[36px] p-6 sm:p-10">
          <div className="text-center">
            <div className="mb-3 text-[12px] text-theme-muted tracking-[0.4em]">
              ENTER THE VIBE
            </div>
            <h1
              className="vhs-tear vhs-tear-strong glow-text jp-art text-4xl text-theme leading-none sm:text-5xl"
              data-text="ノリ"
            >
              ノリ
            </h1>
            <p className="mt-3 text-sm text-theme-muted sm:text-base">
              Shared music rooms for neon nights.
            </p>
            <p className="jp-art mt-2 text-theme-subtle text-xs">音楽を共有</p>
          </div>

          <div className="mt-8 space-y-5">
            <div className="panel-surface rounded-[24px] p-6">
              <label className="mb-3 block font-display text-[10px] text-theme-muted tracking-[0.3em]">
                ROOM NAME
              </label>
              <input
                type="text"
                placeholder="Enter Room Name..."
                value={roomCode}
                onChange={(e) => setRoomCode(e.target.value.toLowerCase())}
                onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
                disabled={isValidating}
                className="w-full rounded-2xl border border-theme bg-theme-surface px-4 py-4 font-mono text-base text-theme tracking-widest placeholder:text-theme-subtle focus:border-secondary focus:outline-hidden focus:ring-2 focus:ring-secondary/30 disabled:cursor-not-allowed disabled:opacity-60"
                maxLength={20}
              />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Link
                to="/rooms/create"
                className="group flex cursor-pointer items-center justify-center gap-3 rounded-2xl border border-primary/50 bg-primary/95 px-6 py-4 font-display text-sm text-white shadow-[0_0_28px_rgba(255,46,151,0.45)] transition-all hover:-translate-y-0.5 hover:bg-primary"
              >
                Start a Session
                <span className="inline-flex h-8 w-8 items-center justify-center rounded-full bg-white/25 text-white">
                  +
                </span>
              </Link>
              <button
                onClick={handleJoinRoom}
                disabled={!roomCode.trim() || isValidating}
                className="flex cursor-pointer items-center justify-center gap-3 rounded-2xl border border-secondary/50 bg-secondary/85 px-6 py-4 font-display text-sm text-white shadow-[0_0_26px_rgba(0,217,255,0.4)] transition-all hover:-translate-y-0.5 hover:bg-secondary disabled:cursor-not-allowed disabled:bg-theme-surface disabled:text-theme-subtle"
              >
                {isValidating ? (
                  <span className="flex items-center gap-2">
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white" />
                    Checking...
                  </span>
                ) : (
                  'Join Room'
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
