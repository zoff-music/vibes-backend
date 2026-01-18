import React, { useRef, useEffect, useState, useCallback } from 'react';
import { View, StyleSheet, ActivityIndicator, Platform } from 'react-native';
import YoutubePlayer, { YoutubeIframeRef } from 'react-native-youtube-iframe';
import ReactPlayer from 'react-player';
import { usePlaybackStore } from '../../stores/playbackStore';

interface Props {
    isVisible?: boolean;
    onEnded?: () => void;
}

export const VideoPlayer: React.FC<Props> = ({ isVisible = true, onEnded }) => {
    const { currentSong, isPlaying } = usePlaybackStore();
    const nativePlayerRef = useRef<YoutubeIframeRef>(null);
    const webPlayerRef = useRef<any>(null);
    const [isReady, setIsReady] = useState(false);
    const [layout, setLayout] = useState({ width: 0, height: 0 });

    // Sync position check loop (every 1s)
    useEffect(() => {
        const interval = setInterval(() => {
            if (isReady && isPlaying) {
                const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
                const targetTime = actualPositionMs / 1000;

                playerRef.current.getCurrentTime().then((currentPlayerTime: number) => {
                    const targetTime = actualPositionMs / 1000;

                    const drift = Math.abs(currentPlayerTime - targetTime);
                    if (drift > 2) {
                        playerRef.current?.seekTo(targetTime, true);
                    }
                } else if (Platform.OS !== 'web' && nativePlayerRef.current) {
                    nativePlayerRef.current.getCurrentTime().then((currentPlayerTime) => {
                        const drift = Math.abs(currentPlayerTime - targetTime);
                        if (drift > 2) {
                            nativePlayerRef.current?.seekTo(targetTime, true);
                        }
                    }).catch(() => { });
                }
            }
        }, 1000);

        return () => clearInterval(interval);
    }, [isReady, isPlaying]);

    const onPlayerReady = useCallback(() => {
        setIsReady(true);
    }, []);

    const onStateChange = useCallback((state: string) => {
        if (state === 'ended') {
            onEnded?.();
        }
    }, [onEnded]);

    const onLayout = useCallback((event: any) => {
        const { width } = event.nativeEvent.layout;
        setLayout({ width, height: width * (9 / 16) });
    }, []);

    if (!currentSong) {
        return null;
    }

    const videoId = currentSong.sourceType === 'youtube' ? currentSong.sourceId : null;

    if (!videoId) return null;

    return (
        <View style={[styles.container, !isVisible && styles.hidden]} onLayout={onLayout}>
            <YoutubePlayer
                ref={playerRef}
                height={layout.height || 220}
                width={layout.width || 320}
                videoId={videoId}
                play={isPlaying}
                onChangeState={onStateChange}
                onReady={onPlayerReady}
                // forceAndroidAutoplay={true}
                // initialPlayerParams={{
                //     controls: false, // We want custom controls or no controls?
                //     modestbranding: true,
                // }}
                // removed webViewProps as it caused issues on web
                contentScale={1}
            />
        </View>
    );
};


const styles = StyleSheet.create({
    container: {
        width: '100%',
        aspectRatio: 16 / 9,
        backgroundColor: '#000',
        borderRadius: 12,
        overflow: 'hidden',
        position: 'relative',
        justifyContent: 'center', // Center the player
    },
    hidden: {
        height: 0,
        opacity: 0,
    },
    loader: {
        ...StyleSheet.absoluteFillObject,
        backgroundColor: 'rgba(0,0,0,0.5)',
        justifyContent: 'center',
        alignItems: 'center',
        zIndex: 1,
    }
});
