import { QueueList, SoundCloudIcon, SpotifyIcon, YouTubeIcon } from '@vibez/ui';
import { QRCodeSVG } from 'qrcode.react';
import React from 'react';
import { useCast } from './CastProvider';
import { PlayerLayer } from './PlayerLayer';

export const ActiveView: React.FC = () => {
  const {
    currentSong,
    queue,
    roomInfo,
    actualPositionMs,
    roomId,
    error,
    apiUrl,
  } = useCast();

  if (!currentSong) return null;

  const joinUrl = `${window.location.origin}/rooms/${roomId}`;
  const upNext = queue.filter((song) => song.id !== currentSong.id);

  return (
    <div className="flex h-screen w-screen flex-row overflow-hidden bg-black font-body">
      {/* Left Column: Player & Current Song Info - FIXED WIDTH */}
      <div className="relative h-full w-[65%] shrink-0 overflow-hidden border-white/10 border-r bg-black shadow-2xl">
        {/* Player Container */}
        <PlayerLayer />

        {/* Info Overlay - TV Optimized */}
        <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black via-black/80 to-transparent px-10 pt-32 pb-10">
          <div className="flex items-end gap-6">
            <div className="relative shrink-0">
              <div className="absolute -inset-1 rounded-2xl bg-gradient-to-tr from-primary to-secondary opacity-30" />
              <img
                src={currentSong.thumbnailUrl}
                alt={currentSong.title}
                className="relative h-28 w-28 rounded-xl border border-white/20 object-cover"
              />
            </div>
            <div className="mb-2 min-w-0 flex-1">
              <h1 className="mb-2 truncate font-display text-4xl text-white leading-tight">
                {currentSong.title}
              </h1>
              <p className="truncate font-light font-sans text-2xl text-white/80">
                {currentSong.artist || 'Unknown Artist'}
              </p>
              {(currentSong.voteCount || 0) > 0 && (
                <div className="mt-3 flex items-center gap-2">
                  <span className="flex items-center gap-2 rounded-full border border-white/5 bg-white/10 px-3 py-1.5 font-medium text-base text-secondary">
                    <span className="text-lg">🔥</span> {currentSong.voteCount}{' '}
                    votes
                  </span>
                </div>
              )}
            </div>

            {/* Source Icon on Right */}
            <div className="mb-2 flex shrink-0 items-center justify-center pr-4 pb-1">
              {currentSong.sourceType === 'spotify' ? (
                <SpotifyIcon className="h-10 w-10 text-white/50" />
              ) : currentSong.sourceType === 'soundcloud' ? (
                <SoundCloudIcon className="h-10 w-10 text-white/50" />
              ) : (
                <YouTubeIcon className="h-10 w-10 text-white/50" />
              )}
            </div>
          </div>

          {/* Progress Bar */}
          <div className="mt-8">
            <div className="mb-3 flex justify-between font-mono text-lg text-white/60">
              <span>
                {Math.floor(actualPositionMs / 60000)}:
                {String(Math.floor((actualPositionMs / 1000) % 60)).padStart(
                  2,
                  '0',
                )}
              </span>
              <span>
                {Math.floor((currentSong.duration || 0) / 60)}:
                {String(Math.floor((currentSong.duration || 0) % 60)).padStart(
                  2,
                  '0',
                )}
              </span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-white/20">
              <div
                className="h-full bg-gradient-to-r from-primary to-secondary"
                style={{
                  width: `${Math.min(
                    (actualPositionMs / ((currentSong.duration || 1) * 1000)) *
                      100,
                    100,
                  )}%`,
                }}
              />
            </div>
          </div>
        </div>
      </div>

      {/* Right Column: Up Next Queue */}
      <div className="flex h-full flex-1 flex-col bg-[#111111] p-10">
        <div className="mb-8 flex items-center justify-between border-white/10 border-b pb-6">
          <div>
            <h2 className="font-display text-sm text-text-muted uppercase tracking-[0.4em]">
              Up Next
            </h2>
          </div>
          {roomInfo && (
            <div className="text-right">
              {/* <h3 className="font-medium text-theme text-xl">
                {roomInfo.name}
              </h3> */}
              <div className="mt-1 flex items-center justify-end gap-2 text-sm text-theme-muted">
                <span className="h-2 w-2 rounded-full bg-green-500"></span>
                {roomInfo.participantCount} listening
              </div>
            </div>
          )}
        </div>

        <div className="custom-scrollbar flex-1 overflow-y-auto pr-2">
          <QueueList songs={upNext} roomId={roomId || undefined} />
        </div>

        <div className="mt-8 rounded-3xl border border-white/10 bg-white/5 p-6 text-center">
          <div className="flex items-center justify-center gap-8">
            <div className="inline-flex items-center justify-center rounded-2xl bg-white p-3 shadow-lg">
              <QRCodeSVG
                value={joinUrl}
                size={140}
                bgColor="#ffffff"
                fgColor="#000000"
                level="M"
              />
            </div>
            <div className="text-left">
              <p className="mb-2 font-display text-2xl text-theme">
                Join the Party
              </p>
              <p className="max-w-[200px] text-lg text-theme-muted leading-snug">
                Scan to add songs & vote from your phone
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Debug Overlay - Always visible for now until fixed */}
      <div className="pointer-events-none absolute top-0 right-0 z-50 border-green-900 border-b border-l bg-black/80 p-4 font-mono text-green-400 text-xs">
        <div>Room: {roomId || 'NULL'}</div>
        <div>Err: {error || 'None'}</div>
        <div>API: {apiUrl}</div>
        <div>
          Q: {queue.length} | Next: {upNext.length}
        </div>
        <div>
          Song: {currentSong?.id} | {currentSong?.title}
        </div>
        <div>Q IDs: {queue.map((s) => s.id).join(', ')}</div>
      </div>
    </div>
  );
};
