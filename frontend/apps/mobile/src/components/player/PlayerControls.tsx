import React from 'react';
import { View, StyleSheet, TouchableOpacity } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { usePlayback } from '../../hooks/usePlayback';
import { Text } from '../ui/Text';

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
        <View style={styles.container}>
            {/* Progress Bar */}
            <View style={styles.progressContainer}>
                <View style={styles.progressBarBg}>
                    <View style={[styles.progressBarFill, { width: `${progress * 100}%` }]} />
                </View>
                <View style={styles.timeContainer}>
                    <Text size="xs" color="muted">{formatTime(actualPositionMs)}</Text>
                    <Text size="xs" color="muted">{currentSong ? formatTime(currentSong.duration * 1000) : '0:00'}</Text>
                </View>
            </View>

            {/* Main Controls */}
            <View style={styles.controlsRow}>
                <TouchableOpacity style={styles.secondaryButton}>
                    <Ionicons name="shuffle-outline" size={24} color="#a1a1aa" />
                </TouchableOpacity>

                <View style={styles.centerControls}>
                    <TouchableOpacity onPress={isPlaying ? pause : play} style={styles.playButton}>
                        <Ionicons
                            name={isPlaying ? "pause" : "play"}
                            size={32}
                            color="#fff"
                            style={!isPlaying ? { marginLeft: 4 } : {}}
                        />
                    </TouchableOpacity>
                </View>

                <TouchableOpacity onPress={skip} style={styles.secondaryButton}>
                    <Ionicons name="play-skip-forward" size={24} color="#fff" />
                </TouchableOpacity>
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        width: '100%',
        paddingHorizontal: 20,
        paddingVertical: 10,
    },
    progressContainer: {
        marginBottom: 20,
    },
    progressBarBg: {
        height: 4,
        backgroundColor: '#333',
        borderRadius: 2,
        overflow: 'hidden',
    },
    progressBarFill: {
        height: '100%',
        backgroundColor: '#a855f7',
    },
    timeContainer: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        marginTop: 8,
    },
    controlsRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: 40,
    },
    centerControls: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    playButton: {
        width: 64,
        height: 64,
        borderRadius: 32,
        backgroundColor: '#a855f7',
        alignItems: 'center',
        justifyContent: 'center',
        shadowColor: '#a855f7',
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.3,
        shadowRadius: 8,
        elevation: 5,
    },
    secondaryButton: {
        padding: 10,
    },
});
