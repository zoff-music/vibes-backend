import { View, Text, Pressable, TextInput, StyleSheet } from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';

// Theme colors
const colors = {
    background: '#0a0a0a',
    surface: '#141414',
    surfaceElevated: '#1c1c1c',
    primary: '#a855f7',
    primaryMuted: '#7c3aed',
    text: '#fafafa',
    textMuted: '#a1a1aa',
    textInverse: '#0a0a0a',
};

export default function Home() {
    const [roomCode, setRoomCode] = useState('');

    const handleCreateRoom = () => {
        router.push('/room/create');
    };

    const handleJoinRoom = () => {
        if (roomCode.trim()) {
            router.push(`/room/${roomCode.trim()}`);
        }
    };

    return (
        <View style={styles.container}>
            <View style={styles.header}>
                <Text style={styles.title}>Vibez</Text>
                <Text style={styles.subtitle}>Shared music queue for everyone</Text>
            </View>

            <View style={styles.actions}>
                <Pressable style={styles.primaryButton} onPress={handleCreateRoom}>
                    <Text style={styles.primaryButtonText}>Create Room</Text>
                </Pressable>

                <View style={styles.divider}>
                    <View style={styles.dividerLine} />
                    <Text style={styles.dividerText}>or join existing</Text>
                    <View style={styles.dividerLine} />
                </View>

                <View style={styles.joinSection}>
                    <TextInput
                        style={styles.input}
                        placeholder="Enter room code"
                        placeholderTextColor="#71717a"
                        value={roomCode}
                        onChangeText={setRoomCode}
                        autoCapitalize="none"
                        autoCorrect={false}
                    />
                    <Pressable
                        style={[styles.secondaryButton, !roomCode.trim() && styles.buttonDisabled]}
                        onPress={handleJoinRoom}
                        disabled={!roomCode.trim()}
                    >
                        <Text style={styles.secondaryButtonText}>Join</Text>
                    </Pressable>
                </View>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background,
        paddingHorizontal: 24,
        justifyContent: 'center',
    },
    header: {
        alignItems: 'center',
        marginBottom: 48,
    },
    title: {
        fontSize: 48,
        fontWeight: '700',
        color: colors.primary,
        marginBottom: 8,
    },
    subtitle: {
        fontSize: 16,
        color: colors.textMuted,
    },
    actions: {
        gap: 24,
    },
    primaryButton: {
        backgroundColor: colors.primary,
        paddingVertical: 16,
        borderRadius: 12,
        alignItems: 'center',
    },
    primaryButtonText: {
        color: colors.textInverse,
        fontSize: 18,
        fontWeight: '600',
    },
    divider: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 16,
    },
    dividerLine: {
        flex: 1,
        height: 1,
        backgroundColor: colors.surfaceElevated,
    },
    dividerText: {
        color: colors.textMuted,
        fontSize: 14,
    },
    joinSection: {
        flexDirection: 'row',
        gap: 8,
    },
    input: {
        flex: 1,
        backgroundColor: colors.surface,
        borderRadius: 8,
        paddingHorizontal: 16,
        paddingVertical: 16,
        color: colors.text,
        fontSize: 16,
    },
    secondaryButton: {
        backgroundColor: colors.surface,
        paddingHorizontal: 24,
        paddingVertical: 16,
        borderRadius: 8,
        justifyContent: 'center',
    },
    secondaryButtonText: {
        color: colors.text,
        fontSize: 16,
        fontWeight: '500',
    },
    buttonDisabled: {
        opacity: 0.5,
    },
});
