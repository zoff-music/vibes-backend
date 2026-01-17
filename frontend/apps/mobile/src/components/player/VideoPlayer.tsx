import React, { useRef, useEffect, useState } from 'react';
import { View, StyleSheet, ActivityIndicator } from 'react-native';
import ReactPlayer from 'react-player';
import { usePlaybackStore } from '../../stores/playbackStore';

interface Props {
    isVisible?: boolean;
    onEnded?: () => void;
}

export const VideoPlayer: React.FC<Props> = ({ isVisible = true, onEnded }) => {
    const { currentSong, isPlaying, actualPositionMs, setPlaybackState } = usePlaybackStore();
    const playerRef = useRef<any>(null);
    const [isReady, setIsReady] = useState(false);

    // Sync position if drift > 2 seconds
    useEffect(() => {
        if (isReady && playerRef.current && isPlaying) {
            const currentPlayerTime = playerRef.current.getCurrentTime();
            const targetTime = actualPositionMs / 1000;

            const drift = Math.abs(currentPlayerTime - targetTime);
            if (drift > 2) {
                playerRef.current.seekTo(targetTime, 'seconds');
            }
        }
    }, [actualPositionMs, isReady, isPlaying]);

    if (!currentSong) {
        return null;
    }

    const videoUrl = currentSong.sourceType === 'youtube'
        ? `https://www.youtube.com/watch?v=${currentSong.sourceId}`
        : '';

    if (!videoUrl) return null;

    const Player = ReactPlayer as any;

    return (
        <View style={[styles.container, !isVisible && styles.hidden]}>
            {!isReady && (
                <View style={styles.loader}>
                    <ActivityIndicator size="large" color="#a855f7" />
                </View>
            )}
            <Player
                ref={playerRef}
                url={videoUrl}
                playing={isPlaying}
                controls={false}
                width="100%"
                height="100%"
                onReady={() => setIsReady(true)}
                onEnded={onEnded}
                onError={(e: any) => console.error('ReactPlayer error:', e)}
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
