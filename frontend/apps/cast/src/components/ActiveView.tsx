import { QueueList } from '@vibez/ui';
import { QRCodeSVG } from 'qrcode.react';
import React from 'react';
import { useCast } from './CastProvider';
import { PlayerLayer } from './PlayerLayer';

export const ActiveView: React.FC = () => {
  const { currentSong, queue, roomInfo, actualPositionMs } = useCast();

  if (!currentSong) return null;

  const roomId = new URLSearchParams(window.location.search).get('roomId')!;
  const joinUrl = `${window.location.origin}/rooms/${roomId}`;
  const upNext = queue.filter((song) => song.id !== currentSong.id);

  return (
    <div className="flex h-full w-full flex-col lg:flex-row">
      {/* Left Column: Player & Current Song Info */}
      <div className="relative h-[55%] w-full overflow-hidden border-theme-subtle border-b bg-black/40 shadow-2xl lg:h-full lg:w-[65%] lg:border-r lg:border-b-0">
        {/* Player Container */}
        <PlayerLayer />

        {/* Info Overlay */}
        <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/90 via-black/60 to-transparent px-6 pt-20 pb-8 lg:px-12 lg:pt-32 lg:pb-12">
          <div className="flex items-end gap-4 lg:gap-8">
            <div className="relative shrink-0">
              <div className="absolute -inset-1 animate-pulse rounded-2xl bg-gradient-to-tr from-primary to-secondary opacity-50 blur-lg" />
              <img
                src={currentSong.thumbnailUrl}
                alt={currentSong.title}
                className="relative h-16 w-16 rounded-xl border border-white/20 object-cover shadow-2xl lg:h-40 lg:w-40 lg:rounded-2xl"
              />
            </div>
            <div className="mb-2 min-w-0 flex-1">
              <h1 className="mb-2 truncate font-display text-white text-xl drop-shadow-lg lg:text-3xl">
                {currentSong.title}
              </h1>
              <p className="truncate font-sans text-sm text-white/80 lg:text-2xl">
                {currentSong.artist || 'Unknown Artist'}
              </p>
              {(currentSong.voteCount || 0) > 0 && (
                <div className="mt-3 flex items-center gap-2">
                  <span className="flex items-center gap-1 rounded-full bg-white/10 px-3 py-1 font-medium text-secondary text-sm backdrop-blur-md">
                    Contains {currentSong.voteCount} votes
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Progress Bar */}
          <div className="mt-6 lg:mt-8">
            <div className="mb-2 flex justify-between font-mono text-white/60 text-xs">
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
            <div className="h-1.5 w-full overflow-hidden rounded-full bg-white/20">
              <div
                className="h-full bg-gradient-to-r from-primary to-secondary transition-all duration-1000 ease-linear"
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
      <div className="cast-queue flex h-[45%] w-full flex-col bg-theme-surface/95 p-6 backdrop-blur-xl lg:h-full lg:w-[35%] lg:p-10">
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="font-display text-[10px] text-theme-muted tracking-[0.3em]">
              Up Next ({upNext.length})
            </h2>
          </div>
          {roomInfo && (
            <div className="text-right">
              <h3 className="font-medium text-theme">{roomInfo.name}</h3>
              <p className="text-theme-muted text-xs">
                {roomInfo.participantCount} listening
              </p>
            </div>
          )}
        </div>

        <div className="mask-linear -mr-2 flex-1 overflow-y-auto pr-2">
          <QueueList songs={upNext} roomId={roomId} />
        </div>

        <div className="mt-6 rounded-2xl border border-theme bg-theme-surface p-5 text-center">
          <div className="mx-auto mb-4 inline-flex items-center justify-center rounded-2xl border border-theme bg-white p-3">
            <QRCodeSVG
              value={joinUrl}
              size={120}
              bgColor="#ffffff"
              fgColor="#2a1840"
              level="H"
            />
          </div>
          <p className="mb-1 font-display text-base text-theme">
            Join the Party
          </p>
          <p className="text-theme-muted text-xs">Scan to add songs & vote</p>
        </div>
      </div>
    </div>
  );
};
