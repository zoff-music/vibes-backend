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
                <div className="relative h-2 bg-surfaceElevated rounded-full overflow-hidden">
                    <div
                        className="absolute inset-y-0 left-0 bg-gradient-to-r from-primary to-secondary transition-all duration-200 ease-out"
                        style={{ width: `${progress * 100}%` }}
                    />
                    {/* Glow effect */}
                    <div
                        className="absolute inset-y-0 left-0 bg-gradient-to-r from-primary to-secondary opacity-50 blur-sm transition-all duration-200 ease-out"
                        style={{ width: `${progress * 100}%` }}
                    />
                </div>
                <div className="flex justify-between mt-3 px-1">
                    <span className="text-xs font-medium text-text-muted tabular-nums">
                        {formatTime(actualPositionMs)}
                    </span>
                    <span className="text-xs font-medium text-text-subtle tabular-nums">
                        {currentSong ? formatTime(currentSong.duration * 1000) : '0:00'}
                    </span>
                </div>
            </div>

            {/* Main Controls */}
            <div className="flex items-center justify-center gap-4">
                {/* Play/Pause Button */}
                <button
                    onClick={isPlaying ? pause : play}
                    disabled={!currentSong}
                    className="relative group disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    <div className="absolute inset-0 bg-gradient-to-br from-primary to-secondary rounded-full blur-xl opacity-30 group-hover:opacity-50 transition-opacity" />
                    <div className="relative w-16 h-16 rounded-full bg-gradient-to-br from-primary to-secondary flex items-center justify-center hover:scale-105 active:scale-95 transition-transform shadow-glow">
                        {isPlaying ? (
                            <svg className="w-7 h-7 text-white" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                            </svg>
                        ) : (
                            <svg className="w-7 h-7 text-white ml-0.5" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M8 5v14l11-7z" />
                            </svg>
                        )}
                    </div>
                </button>

                {/* Skip Button */}
                <button
                    onClick={skip}
                    disabled={!currentSong}
                    className="glass p-3.5 rounded-full hover:bg-surfaceHover active:scale-95 transition-all disabled:opacity-30 disabled:cursor-not-allowed group"
                    title="Skip"
                >
                    <svg className="w-5 h-5 text-text-muted group-hover:text-white transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>
            </div>
        </div>
    );
};
