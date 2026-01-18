import { useRef, useEffect, useState, useCallback } from 'react';
import YouTube, { YouTubeProps } from 'react-youtube';
import { usePlaybackStore } from '../../stores/playbackStore';

interface Props {
    isVisible?: boolean;
    onEnded?: () => void;
}

export const VideoPlayer: React.FC<Props> = ({ isVisible = true, onEnded }) => {
    const { currentSong, isPlaying } = usePlaybackStore();
    const playerRef = useRef<any>(null);
    const [isReady, setIsReady] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Debug: Log current song changes and reset ready state
    useEffect(() => {
        console.log('[VideoPlayer] currentSong changed:', {
            currentSong,
            hasSong: !!currentSong,
            sourceType: currentSong?.sourceType,
            sourceId: currentSong?.sourceId,
            title: currentSong?.title,
        });
        // Reset ready state when song changes
        setIsReady(false);
        setError(null);
    }, [currentSong?.id]);

    // Debug: Log playing state changes
    useEffect(() => {
        console.log('[VideoPlayer] isPlaying changed:', isPlaying);
    }, [isPlaying]);

    // Debug: Log ready state changes
    useEffect(() => {
        console.log('[VideoPlayer] isReady changed:', isReady);
    }, [isReady]);

    // Sync position check loop (every 1s)
    useEffect(() => {
        const interval = setInterval(() => {
            if (!isReady || !playerRef.current) {
                return;
            }

            try {
                const player = playerRef.current;
                const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
                const targetTime = actualPositionMs / 1000;

                const currentTime = player.getCurrentTime();
                const drift = Math.abs(currentTime - targetTime);
                
                if (drift > 2) {
                    console.log('[VideoPlayer] Seeking due to drift:', { currentTime, targetTime, drift });
                    player.seekTo(targetTime, true);
                }
            } catch (err) {
                console.warn('[VideoPlayer] Error syncing position:', err);
            }
        }, 1000);

        return () => clearInterval(interval);
    }, [isReady, isPlaying]);

    // Control playback based on isPlaying state
    useEffect(() => {
        if (!isReady || !playerRef.current) return;

        try {
            const player = playerRef.current;
            const state = player.getPlayerState();
            
            // YouTube PlayerState: -1 (unstarted), 0 (ended), 1 (playing), 2 (paused), 3 (buffering), 5 (cued)
            if (isPlaying && state !== 1) {
                player.playVideo();
            } else if (!isPlaying && state === 1) {
                player.pauseVideo();
            }
        } catch (err) {
            console.warn('[VideoPlayer] Error controlling playback:', err);
        }
    }, [isReady, isPlaying]);

    const handleReady = useCallback((event: { target: { seekTo: (seconds: number, allowSeekAhead?: boolean) => void; getCurrentTime: () => number; playVideo: () => void; pauseVideo: () => void; getPlayerState: () => number } }) => {
        console.log('[VideoPlayer] Player ready');
        playerRef.current = event.target;
        setIsReady(true);
        setError(null);

        // Sync initial position
        const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
        if (actualPositionMs > 0) {
            const targetTime = actualPositionMs / 1000;
            event.target.seekTo(targetTime, true);
        }
    }, []);

    const handleStateChange = useCallback((event: { data: number }) => {
        const state = event.data;
        console.log('[VideoPlayer] State changed:', state);
        
        // Update isReady when player becomes ready
        if (state === 1 || state === 3) {
            setIsReady(true);
        }
    }, []);

    const handleEnd = useCallback(() => {
        console.log('[VideoPlayer] Video ended');
        onEnded?.();
    }, [onEnded]);

    const handleError = useCallback((event: unknown) => {
        console.error('[VideoPlayer] Player error:', event);
        setError('Failed to load video');
    }, []);

    if (!currentSong || !isVisible) {
        console.log('[VideoPlayer] Not rendering - no song or not visible:', { 
            hasSong: !!currentSong, 
            isVisible 
        });
        return null;
    }

    const videoId = currentSong.sourceType === 'youtube' ? currentSong.sourceId : null;

    if (!videoId) {
        console.warn('[VideoPlayer] No video ID found:', {
            sourceType: currentSong.sourceType,
            sourceId: currentSong.sourceId,
        });
        return (
            <div className="w-full aspect-video bg-black rounded-xl overflow-hidden relative flex items-center justify-center">
                <p className="text-text-muted">Unsupported source type: {currentSong.sourceType}</p>
            </div>
        );
    }

    console.log('[VideoPlayer] Rendering player with videoId:', videoId);

    const opts: YouTubeProps['opts'] = {
        height: '100%',
        width: '100%',
        playerVars: {
            autoplay: 0,
            controls: 0,
            rel: 0,
            modestbranding: 1,
            enablejsapi: 1,
            playsinline: 1,
        },
    };

    return (
        <div 
            className={`w-full bg-black rounded-xl overflow-hidden relative ${!isVisible ? 'hidden' : ''}`}
            style={{ 
                aspectRatio: '16/9',
                minHeight: '200px',
            }}
        >
            {error && (
                <div className="absolute inset-0 flex items-center justify-center bg-black/90 z-20">
                    <div className="text-center px-4">
                        <p className="text-error text-sm mb-2">Error loading video</p>
                        <p className="text-text-muted text-xs">{error}</p>
                        <p className="text-text-muted text-xs mt-2">Video ID: {videoId}</p>
                    </div>
                </div>
            )}
            <div className="absolute inset-0 w-full h-full">
                <YouTube
                    key={videoId}
                    videoId={videoId}
                    opts={opts}
                    onReady={handleReady as YouTubeProps['onReady']}
                    onStateChange={handleStateChange as YouTubeProps['onStateChange']}
                    onEnd={handleEnd as YouTubeProps['onEnd']}
                    onError={handleError as YouTubeProps['onError']}
                    className="w-full h-full"
                />
            </div>
            {!isReady && !error && !isPlaying && (
                <div className="absolute inset-0 flex items-center justify-center bg-black/70 z-10">
                    <div className="text-center">
                        <p className="text-text-muted text-sm mb-1">Loading video...</p>
                        <p className="text-text-muted text-xs">{currentSong.title}</p>
                    </div>
                </div>
            )}
        </div>
    );
};
