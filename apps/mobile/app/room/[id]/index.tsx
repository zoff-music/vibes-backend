import { View, Text, Pressable, StyleSheet } from 'react-native';
import { useLocalSearchParams, router } from 'expo-router';

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

    return (
        <View style={styles.container}>
            <View style={styles.header}>
                <Pressable onPress={() => router.back()}>
                    <Text style={styles.backButton}>← Back</Text>
                </Pressable>
            </View>

            <View style={styles.nowPlaying}>
                <View style={styles.artwork} />
                <Text style={styles.title}>No song playing</Text>
                <Text style={styles.artist}>Add songs to get started</Text>
            </View>

            <View style={styles.controls}>
                <Text style={styles.controlsPlaceholder}>
                    Playback controls will appear here
                </Text>
            </View>

            <View style={styles.queue}>
                <Text style={styles.queueTitle}>Up Next</Text>
                <Text style={styles.emptyQueue}>Queue is empty</Text>
            </View>

            <Text style={styles.roomId}>Room: {id}</Text>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background,
        paddingHorizontal: 24,
        paddingTop: 48,
    },
    header: {
        marginBottom: 24,
    },
    backButton: {
        color: colors.primary,
        fontSize: 16,
    },
    nowPlaying: {
        alignItems: 'center',
        marginBottom: 32,
    },
    artwork: {
        width: 200,
        height: 200,
        backgroundColor: colors.surface,
        borderRadius: 12,
        marginBottom: 24,
    },
    title: {
        fontSize: 24,
        fontWeight: '600',
        color: colors.text,
        marginBottom: 4,
    },
    artist: {
        fontSize: 16,
        color: colors.textMuted,
    },
    controls: {
        alignItems: 'center',
        paddingVertical: 24,
        borderTopWidth: 1,
        borderBottomWidth: 1,
        borderColor: colors.surfaceElevated,
        marginBottom: 24,
    },
    controlsPlaceholder: {
        color: colors.textMuted,
        fontSize: 14,
    },
    queue: {
        flex: 1,
    },
    queueTitle: {
        fontSize: 18,
        fontWeight: '600',
        color: colors.text,
        marginBottom: 16,
    },
    emptyQueue: {
        color: colors.textMuted,
        fontSize: 16,
        textAlign: 'center',
        paddingVertical: 32,
    },
    roomId: {
        color: colors.textMuted,
        fontSize: 12,
        textAlign: 'center',
        paddingVertical: 16,
    },
});
