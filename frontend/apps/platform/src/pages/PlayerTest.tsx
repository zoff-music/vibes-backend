import { useRef, useEffect } from 'react';
import YouTube, { YouTubeProps } from 'react-youtube';

export default function PlayerTest() {
    const videoId = 'LXb3EKWsInQ';
    const playerRef = useRef<any>(null);
    const playAttemptsRef = useRef(0);
    const maxPlayAttempts = 5;

    const opts: YouTubeProps['opts'] = {
        height: '390',
        width: '100%',
        playerVars: {
            autoplay: 1,
            controls: 1,
            rel: 0,
            modestbranding: 1,
            mute: 0,
        },
    };

    const forcePlay = () => {
        if (!playerRef.current) {
            console.log('[PlayerTest] Player ref not available yet');
            return;
        }

        const player = playerRef.current;
        
        try {
            const state = player.getPlayerState();
            console.log('[PlayerTest] Current player state:', state);
            
            // YouTube PlayerState: -1 (unstarted), 0 (ended), 1 (playing), 2 (paused), 3 (buffering), 5 (cued)
            if (state !== 1) {
                console.log('[PlayerTest] Forcing play, attempt:', playAttemptsRef.current + 1);
                player.seekTo(10);
                player.playVideo();
                playAttemptsRef.current += 1;
            } else {
                console.log('[PlayerTest] Already playing');
            }
        } catch (err) {
            console.error('[PlayerTest] Error forcing play:', err);
        }
    };

    useEffect(() => {
        // Set up interval to check and force play if needed
        const interval = setInterval(() => {
            if (playerRef.current && playAttemptsRef.current < maxPlayAttempts) {
                forcePlay();
            } else if (playAttemptsRef.current >= maxPlayAttempts) {
                console.log('[PlayerTest] Max play attempts reached');
                clearInterval(interval);
            }
        }, 1000);

        return () => clearInterval(interval);
    }, []);

    const onReady: YouTubeProps['onReady'] = (event) => {
        console.log('[PlayerTest] YouTube player ready');
        playerRef.current = event.target;
        
        // Force play immediately
        setTimeout(() => {
            forcePlay();
        }, 500);
    };

    const onStateChange: YouTubeProps['onStateChange'] = (event) => {
        const state = event.data;
        console.log('[PlayerTest] State changed:', state);
        
        // YouTube PlayerState enum: -1 (unstarted), 0 (ended), 1 (playing), 2 (paused), 3 (buffering), 5 (cued)
        if (state === -1 || state === 5) {
            // Unstarted or cued - try to play
            setTimeout(() => {
                forcePlay();
            }, 500);
        }
    };

    const onPlay = () => {
        console.log('[PlayerTest] Video started playing');
        playAttemptsRef.current = 0; // Reset attempts on successful play
    };

    const onPause = () => {
        console.log('[PlayerTest] Video paused');
    };

    const onEnd = () => {
        console.log('[PlayerTest] Video ended');
    };

    const onError: YouTubeProps['onError'] = (event) => {
        console.error('[PlayerTest] Player error:', event);
    };

    return (
        <div className="min-h-screen bg-background text-text flex flex-col items-center justify-center p-6">
            <div className="w-full max-w-3xl">
                <YouTube
                    videoId={videoId}
                    opts={opts}
                    onReady={onReady}
                    onStateChange={onStateChange}
                    onPlay={onPlay}
                    onPause={onPause}
                    onEnd={onEnd}
                    onError={onError}
                />
            </div>
        </div>
    );
}
