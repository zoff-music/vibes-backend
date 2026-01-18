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
        <div className="w-full px-5 py-2.5">
            {/* Progress Bar */}
            <div className="mb-5">
                <div className="h-1 bg-zinc-800 rounded-full overflow-hidden">
                    <div
                        className="h-full bg-primary transition-all"
                        style={{ width: `${progress * 100}%` }}
                    />
                </div>
                <div className="flex justify-between mt-2">
                    <span className="text-xs text-text-muted">{formatTime(actualPositionMs)}</span>
                    <span className="text-xs text-text-muted">
                        {currentSong ? formatTime(currentSong.duration * 1000) : '0:00'}
                    </span>
                </div>
            </div>

            {/* Main Controls */}
            <div className="flex items-center justify-between px-10">
                <button className="p-2.5 text-text-muted hover:text-text transition-colors">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        className="h-6 w-6"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                    >
                        <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
                        />
                    </svg>
                </button>

                <div className="flex items-center">
                    <button
                        onClick={isPlaying ? pause : play}
                        className="w-16 h-16 rounded-full bg-primary flex items-center justify-center hover:opacity-90 transition-opacity shadow-lg shadow-primary/30"
                    >
                        {isPlaying ? (
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                className="h-8 w-8 text-white"
                                fill="currentColor"
                                viewBox="0 0 24 24"
                            >
                                <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                            </svg>
                        ) : (
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                className="h-8 w-8 text-white ml-1"
                                fill="currentColor"
                                viewBox="0 0 24 24"
                            >
                                <path d="M8 5v14l11-7z" />
                            </svg>
                        )}
                    </button>
                </div>

                <button
                    onClick={skip}
                    className="p-2.5 text-text hover:text-primary transition-colors"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        className="h-6 w-6"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                    >
                        <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M9 5l7 7-7 7"
                        />
                        <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M16 5l7 7-7 7"
                        />
                    </svg>
                </button>
            </div>
        </div>
    );
};
