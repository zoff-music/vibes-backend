import { QueueList } from '@vibez/ui';
import React from 'react';
import { useCast } from './CastProvider';
import { PlayerLayer } from './PlayerLayer';

export const ActiveView: React.FC = () => {
    const { currentSong, queue, roomInfo, actualPositionMs } = useCast();

    if (!currentSong) return null;

    const roomId = new URLSearchParams(window.location.search).get('roomId')!;

    return (
        <div className="flex h-full w-full">
            {/* Left Column: Player & Current Song Info (65%) */}
            <div className="relative h-full w-[65%] overflow-hidden border-r border-theme-subtle bg-black/40 shadow-2xl">
                {/* Player Container */}
                <PlayerLayer />

                {/* Info Overlay */}
                <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/90 via-black/60 to-transparent pt-32 pb-12 pl-12 pr-12">
                    <div className="flex items-end gap-8">
                        <div className="relative shrink-0">
                            <div className="absolute -inset-1 animate-pulse rounded-2xl bg-gradient-to-tr from-primary to-secondary opacity-50 blur-lg" />
                            <img
                                src={currentSong.thumbnailUrl}
                                alt={currentSong.title}
                                className="relative h-40 w-40 rounded-2xl border border-white/20 object-cover shadow-2xl"
                            />
                        </div>
                        <div className="flex-1 min-w-0 mb-2">
                            <h1 className="mb-2 truncate font-display text-5xl text-white drop-shadow-lg">
                                {currentSong.title}
                            </h1>
                            <p className="truncate font-sans text-2xl text-white/80">
                                {currentSong.artist || 'Unknown Artist'}
                            </p>
                            {(currentSong.voteCount || 0) > 0 && (
                                <div className="mt-3 flex items-center gap-2">
                                    <span className="flex items-center gap-1 rounded-full bg-white/10 px-3 py-1 font-medium text-sm text-secondary backdrop-blur-md">
                                        Contains {currentSong.voteCount} votes
                                    </span>
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Progress Bar */}
                    <div className="mt-8">
                        <div className="mb-2 flex justify-between text-xs font-mono text-white/60">
                            <span>
                                {Math.floor(actualPositionMs / 60000)}:
                                {String(
                                    Math.floor((actualPositionMs / 1000) % 60),
                                ).padStart(2, '0')}
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

            {/* Right Column: Up Next Queue (35%) */}
            <div className="flex w-[35%] flex-col bg-theme-surface/95 p-10 backdrop-blur-xl">
                <div className="mb-8 flex items-center justify-between">
                    <div>
                        <h2 className="font-display text-2xl text-theme">Up Next</h2>
                        <p className="text-sm text-theme-muted">
                            {queue.length} songs in queue
                        </p>
                    </div>
                    {roomInfo && (
                        <div className="text-right">
                            <h3 className="font-medium text-theme">{roomInfo.name}</h3>
                            <p className="text-xs text-theme-muted">
                                {roomInfo.participantCount} listening
                            </p>
                        </div>
                    )}
                </div>

                <div className="flex-1 overflow-y-auto pr-2 -mr-2 mask-linear">
                    <QueueList
                        songs={queue.slice(0, 5)}
                        roomId={roomId}
                    />

                    {queue.length > 5 && (
                        <div className="mt-4 border-t border-theme-subtle py-4 text-center">
                            <p className="text-sm text-theme-muted">
                                + {queue.length - 5} more songs
                            </p>
                        </div>
                    )}
                </div>

                {/* QR Code / Join Info could go here */}
                <div className="mt-8 rounded-2xl border border-theme bg-white/5 p-6 text-center">
                    <p className="mb-2 font-display text-lg text-theme">
                        Join the Party
                    </p>
                    <p className="text-sm text-theme-muted">
                        Scan to add songs & vote
                    </p>
                </div>
            </div>
        </div>
    );
};
