import React from 'react';
import { usePlayback } from '../../hooks/usePlayback';

interface Props {
    roomId: string;
}

export const PlayerControls: React.FC<Props> = ({ roomId }) => {
    const { isPlaying, play, pause, skip, actualPositionMs, currentSong } = usePlayback(roomId);

    const formatTime = (ms: number) => {
        const totalSeconds = Math.floor(ms / 1000);
        const minutes = Math.floor(totalSeconds / 60);
        const seconds = totalSeconds % 60;
        return `${minutes}:${seconds.toString().padStart(2, '0')}`;
    };

    const progress = currentSong ? (actualPositionMs / (currentSong.duration * 1000)) : 0;

    return (
        <div className="w-full space-y-6">
            {/* Progress Bar */}
            <div>
                <div className="relative h-3 bg-ink/10 rounded-full overflow-hidden border-2 border-ink/20">
                    <div
                        className="absolute inset-y-0 left-0 bg-gradient-to-r from-primary via-secondary to-accent transition-all duration-200 ease-out"
                        style={{ width: `${progress * 100}%` }}
                    />
                </div>
                <div className="flex justify-between mt-3 px-1">
                    <span className="text-xs font-bold text-ink/70 tabular-nums tracking-wider">
                        {formatTime(actualPositionMs)}
                    </span>
                    <span className="text-xs font-bold text-ink/50 tabular-nums tracking-wider">
                        {currentSong ? formatTime(currentSong.duration * 1000) : '0:00'}
                    </span>
                </div>
            </div>

            {/* Main Controls */}
            <div className="flex items-center justify-center gap-5">
                {/* Play/Pause Button */}
                <button
                    onClick={isPlaying ? pause : play}
                    disabled={!currentSong}
                    className="relative group disabled:opacity-40 disabled:cursor-not-allowed"
                >
                    <div className="relative w-20 h-20 rounded-2xl bg-primary flex items-center justify-center hover:scale-105 active:scale-95 transition-all shadow-retro-pink hover:shadow-neon-pink border-4 border-white">
                        {isPlaying ? (
                            <svg className="w-9 h-9 text-white" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                            </svg>
                        ) : (
                            <svg className="w-9 h-9 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M8 5v14l11-7z" />
                            </svg>
                        )}
                    </div>
                </button>

                {/* Skip Button */}
                <button
                    onClick={skip}
                    disabled={!currentSong}
                    className="glass p-4 rounded-xl hover:shadow-retro active:scale-95 transition-all disabled:opacity-30 disabled:cursor-not-allowed group border-2 border-ink/10"
                    title="Skip"
                >
                    <svg className="w-6 h-6 text-ink/60 group-hover:text-primary transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>
            </div>
        </div>
    );
};
