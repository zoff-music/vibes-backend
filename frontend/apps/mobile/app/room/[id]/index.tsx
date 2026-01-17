import React, { useEffect, useState } from 'react';
import { View, Pressable, StyleSheet, ScrollView } from 'react-native';
import { useLocalSearchParams, router } from 'expo-router';
import { useRoom } from '../../../src/hooks/useRoom';
import { usePlayback } from '../../../src/hooks/usePlayback';
import { useQueue } from '../../../src/hooks/useQueue';
import { VideoPlayer } from '../../../src/components/player/VideoPlayer';
import { PlayerControls } from '../../../src/components/player/PlayerControls';
import { AddToQueueModal } from '../../../src/components/queue/AddToQueueModal';
import { Text } from '../../../src/components/ui/Text';
import { Button } from '../../../src/components/ui/Button';

const colors = {
    background: '#0a0a0a',
    surface: '#141414',
    surfaceElevated: '#1c1c1c',
    primary: '#a855f7',
    text: '#fafafa',
    textMuted: '#a1a1aa',
};

export default function RoomView() {
    const { id } = useLocalSearchParams<{ id: string }>();
    const { currentSong } = usePlayback(id as string);
    const { songs, fetchQueue } = useQueue(id as string);
    const { room, fetchRoom, isLoading, error, joinRoom, userId } = useRoom(id as string);

    const [isAddModalVisible, setIsAddModalVisible] = useState(false);

    useEffect(() => {
        if (id) {
            fetchRoom();
            fetchQueue();
        }
    }, [id, fetchRoom, fetchQueue]);

    // Auto-join if not already in session
    useEffect(() => {
        if (!id || userId) return;

        const checkJoin = async () => {
            // For now, just join as "Guest"
            await joinRoom("Guest_" + Math.floor(Math.random() * 1000));
        };

        checkJoin();
    }, [id, userId, joinRoom]);

    if (isLoading && !room) {
        return (
            <View style={[styles.container, styles.centered]}>
                <Text>Loading Room...</Text>
            </View>
        );
    }

    if (error) {
        return (
            <View style={[styles.container, styles.centered]}>
                <Text color="primary">Error: {error.message}</Text>
                <Button onPress={() => fetchRoom()} title="Retry" style={{ marginTop: 20 }} />
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <View style={styles.header}>
                <Pressable onPress={() => router.back()}>
                    <Text style={styles.backButton}>← Back</Text>
                </Pressable>
                <Text size="lg" bold>{room?.name || 'Loading...'}</Text>
            </View>

            <ScrollView style={styles.content} showsVerticalScrollIndicator={false}>
                <View style={styles.playerSection}>
                    <VideoPlayer />

                    <View style={styles.nowPlayingInfo}>
                        <Text size="xl" bold numberOfLines={1}>
                            {currentSong?.title || 'No song playing'}
                        </Text>
                        <Text color="muted" size="md">
                            {currentSong?.artist || 'Add songs to get started'}
                        </Text>
                    </View>
                </View>

                <PlayerControls roomId={id as string} />

                <View style={styles.queueHeader}>
                    <Text size="lg" bold>Up Next</Text>
                    <Button
                        onPress={() => setIsAddModalVisible(true)}
                        title="Add Song"
                        variant="ghost"
                        size="sm"
                    />
                </View>

                <View style={styles.queue}>
                    {songs.length === 0 ? (
                        <Text style={styles.emptyQueue}>Queue is empty</Text>
                    ) : (
                        songs.map((song, index) => (
                            <View key={song.id} style={styles.queueItem}>
                                <View style={styles.songIndex}>
                                    <Text color="muted">{index + 1}</Text>
                                </View>
                                <View style={styles.songInfo}>
                                    <Text numberOfLines={1}>{song.title}</Text>
                                    <Text size="xs" color="muted">{song.artist || 'Unknown Artist'}</Text>
                                </View>
                            </View>
                        ))
                    )}
                </View>
            </ScrollView>

            <View style={styles.footer}>
                <Text size="xs" color="muted">Room PIN: {id}</Text>
            </View>

            <AddToQueueModal
                roomId={id as string}
                isVisible={isAddModalVisible}
                onClose={() => setIsAddModalVisible(false)}
            />
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background,
        paddingTop: 48,
    },
    centered: {
        justifyContent: 'center',
        alignItems: 'center',
    },
    header: {
        paddingHorizontal: 24,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        marginBottom: 24,
    },
    backButton: {
        color: colors.primary,
        fontSize: 16,
    },
    content: {
        flex: 1,
    },
    playerSection: {
        paddingHorizontal: 24,
        alignItems: 'center',
        marginBottom: 24,
    },
    nowPlayingInfo: {
        width: '100%',
        marginTop: 16,
        alignItems: 'center',
    },
    queueHeader: {
        paddingHorizontal: 24,
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginTop: 32,
        marginBottom: 16,
    },
    queue: {
        paddingHorizontal: 24,
        paddingBottom: 40,
    },
    emptyQueue: {
        color: colors.textMuted,
        fontSize: 16,
        textAlign: 'center',
        paddingVertical: 32,
    },
    queueItem: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: colors.surface,
        padding: 12,
        borderRadius: 8,
        marginBottom: 8,
    },
    songIndex: {
        width: 30,
    },
    songInfo: {
        flex: 1,
    },
    footer: {
        paddingVertical: 16,
        alignItems: 'center',
        borderTopWidth: 1,
        borderColor: colors.surfaceElevated,
    },
});

