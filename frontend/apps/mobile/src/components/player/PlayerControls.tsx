import React from 'react';
import { usePlayback } from '../../hooks/usePlayback';

interface Props {
    roomId: string;
    hasSongsInQueue?: boolean;
    onAddSong: () => void;
}

export const PlayerControls: React.FC<Props> = ({ roomId, hasSongsInQueue = false, onAddSong }) => {
    const { isPlaying, play, pause, skip, currentSong } = usePlayback(roomId);

    const btnClass = "glass p-4 rounded-xl hover:shadow-retro active:scale-95 transition-all disabled:opacity-30 disabled:cursor-not-allowed group border-2 border-ink/10 flex items-center justify-center h-14";

    return (
        <div className="w-full">
            {/* Main Controls */}
            <div className="flex items-center justify-start gap-4">
                {/* Play/Pause Button */}
                <button
                    onClick={isPlaying ? pause : play}
                    disabled={!currentSong && !hasSongsInQueue}
                    className={`bg-primary shadow-retro-pink hover:shadow-neon-pink border-2 border-white/50 text-white w-14 h-14 rounded-xl hover:shadow-retro active:scale-95 transition-all disabled:opacity-30 disabled:cursor-not-allowed group flex items-center justify-center`}
                >
                    {isPlaying ? (
                        <svg className="w-6 h-6 fill-current" viewBox="0 0 24 24">
                            <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                        </svg>
                    ) : (
                        <svg className="w-6 h-6 fill-current ml-0.5" viewBox="0 0 24 24">
                            <path d="M8 5v14l11-7z" />
                        </svg>
                    )}
                </button>

                {/* Skip Button */}
                <button
                    onClick={skip}
                    disabled={!currentSong}
                    className={`${btnClass} w-14`}
                    title="Skip"
                >
                    <svg className="w-6 h-6 text-ink/60 group-hover:text-primary transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>

                {/* Add Song Button */}
                <button
                    onClick={onAddSong}
                    className={`${btnClass} text-primary hover:border-primary/30 gap-2 px-6 ml-auto`}
                    title="Add Song"
                >
                    <svg className="w-5 h-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                    </svg>
                    <span className="font-black text-ink tracking-wide text-sm whitespace-nowrap">Add Song</span>
                </button>
            </div>
        </div>
    );
};
