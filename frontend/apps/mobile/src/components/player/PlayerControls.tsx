import React, { useState } from 'react';
import { usePlayback } from '../../hooks/usePlayback';
import { useCasting } from '../../hooks/useCasting';
import { DeviceSelector } from '../cast/DeviceSelector';

interface Props {
    roomId: string;
    hasSongsInQueue?: boolean;
    onAddSong: () => void;
}

export const PlayerControls: React.FC<Props> = ({ roomId, hasSongsInQueue = false, onAddSong }) => {
    const { isPlaying, play, pause, skip, currentSong } = usePlayback(roomId);
    const { isConnected, castDeviceName } = useCasting(roomId);
    const [showDeviceSelector, setShowDeviceSelector] = useState(false);

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

                {/* Cast Button */}
                <button
                    onClick={() => setShowDeviceSelector(true)}
                    className={`${btnClass} w-14 ${isConnected ? 'bg-primary/10 border-primary/20 dark:bg-primary/20 dark:border-primary/30' : ''}`}
                    title={isConnected ? `Casting to ${castDeviceName}` : 'Cast'}
                >
                    <svg className={`w-6 h-6 transition-colors ${isConnected ? 'text-primary dark:text-primary-light' : 'text-ink/60 group-hover:text-primary dark:text-dark-text-muted dark:group-hover:text-primary'}`} fill="currentColor" viewBox="0 0 24 24">
                        <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
                        {isConnected && (
                            <circle cx="6" cy="18" r="2" className="fill-current"/>
                        )}
                    </svg>
                </button>

                {/* Add Song Button */}
                <button
                    onClick={onAddSong}
                    className={`${btnClass} text-primary hover:border-primary/30 gap-2 px-6 ml-auto`}
                    title="Add Song"
                >
                    <svg className="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                    </svg>
                    <span className="font-black text-ink tracking-wide text-sm whitespace-nowrap">Add Song</span>
                </button>
            </div>

            {/* Casting Status */}
            {isConnected && castDeviceName && (
                <div className="mt-3 flex items-center justify-center gap-2 text-xs text-primary dark:text-primary-light">
                    <div className="w-2 h-2 bg-primary dark:bg-primary-light rounded-full animate-pulse"></div>
                    <span className="font-medium">Casting to {castDeviceName}</span>
                </div>
            )}

            {/* Device Selector Modal */}
            <DeviceSelector 
                isOpen={showDeviceSelector} 
                onClose={() => setShowDeviceSelector(false)} 
            />
        </div>
    );
};
