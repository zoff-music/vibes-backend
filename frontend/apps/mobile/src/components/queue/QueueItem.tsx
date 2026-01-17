import React from 'react';
import { View, StyleSheet, Image, TouchableOpacity } from 'react-native';
import { Song } from '@vibez/shared';
import { Text } from '../ui/Text';
import { Ionicons } from '@expo/vector-icons';

interface Props {
    song: Song;
    onRemove?: (id: string) => void;
    isAdmin?: boolean;
}

export const QueueItem: React.FC<Props> = ({ song, onRemove, isAdmin }) => {
    const formatDuration = (seconds: number) => {
        const min = Math.floor(seconds / 60);
        const sec = seconds % 60;
        return `${min}:${sec.toString().padStart(2, '0')}`;
    };

    return (
        <View style={styles.container}>
            <Image source={{ uri: song.thumbnailUrl }} style={styles.thumbnail} />

            <View style={styles.content}>
                <Text bold numberOfLines={1}>{song.title}</Text>
                <Text size="sm" color="muted" numberOfLines={1}>
                    {song.artist || 'Unknown Artist'} • {formatDuration(song.duration)}
                </Text>
            </View>

            <View style={styles.actions}>
                {isAdmin && (
                    <TouchableOpacity onPress={() => onRemove?.(song.id)} style={styles.actionButton}>
                        <Ionicons name="trash-outline" size={20} color="#ef4444" />
                    </TouchableOpacity>
                )}
                <Ionicons name="menu-outline" size={20} color="#a1a1aa" />
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flexDirection: 'row',
        alignItems: 'center',
        padding: 12,
        backgroundColor: '#141414',
        borderRadius: 12,
        marginBottom: 8,
    },
    thumbnail: {
        width: 48,
        height: 48,
        borderRadius: 6,
        backgroundColor: '#333',
    },
    content: {
        flex: 1,
        marginLeft: 12,
    },
    actions: {
        flexDirection: 'row',
        alignItems: 'center',
        marginLeft: 12,
    },
    actionButton: {
        padding: 8,
        marginRight: 4,
    },
});
