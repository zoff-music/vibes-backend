import { View, Text, Pressable, TextInput, Switch, ScrollView, StyleSheet, ActivityIndicator, Alert } from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { api } from '../../src/api/client';

const colors = {
    background: '#0a0a0a',
    surface: '#141414',
    primary: '#a855f7',
    primaryMuted: '#7c3aed',
    text: '#fafafa',
    textMuted: '#a1a1aa',
    textInverse: '#0a0a0a',
    error: '#ef4444',
};

const DEFAULT_SETTINGS = {
    skipAllowed: true,
    democraticSkip: true,
    loopQueue: false,
};

export default function CreateRoom() {
    const [name, setName] = useState('');
    const [password, setPassword] = useState('');
    const [settings, setSettings] = useState(DEFAULT_SETTINGS);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleCreate = async () => {
        if (!name.trim() || isLoading) return;

        setIsLoading(true);
        setError(null);

        const [err, room] = await api.post('/rooms', null, {
            name: name.trim(),
            password: password || undefined,
        });

        if (err) {
            console.error('Failed to create room:', err);
            setError(err.message || 'Failed to create room');
            setIsLoading(false);
            return;
        }

        if (room) {
            const createdAt = new Date(room.createdAt);
            const now = new Date();
            // If room was created more than 10 seconds ago, it's an existing room
            const isExisting = now.getTime() - createdAt.getTime() > 10000;

            if (isExisting) {
                Alert.alert('Welcome', 'That room already exists, welcome!');
            }
            
            router.replace(`/room/${room.id}`);
        }
    };

    const updateSetting = <K extends keyof typeof settings>(
        key: K,
        value: (typeof settings)[K]
    ) => {
        setSettings((prev) => ({ ...prev, [key]: value }));
    };

    return (
        <ScrollView style={styles.container} contentContainerStyle={styles.content}>
            <View style={styles.header}>
                <Pressable onPress={() => router.back()}>
                    <Text style={styles.backButton}>← Back</Text>
                </Pressable>
                <Text style={styles.title}>Create Room</Text>
            </View>

            <View style={styles.form}>
                <View style={styles.field}>
                    <Text style={styles.label}>Room Name</Text>
                    <TextInput
                        style={styles.input}
                        placeholder="Friday Night Vibes"
                        placeholderTextColor="#71717a"
                        value={name}
                        onChangeText={setName}
                    />
                </View>

                <View style={styles.field}>
                    <Text style={styles.label}>Admin Password (optional)</Text>
                    <TextInput
                        style={styles.input}
                        placeholder="Leave empty for no password"
                        placeholderTextColor="#71717a"
                        value={password}
                        onChangeText={setPassword}
                        secureTextEntry
                    />
                </View>

                <View style={styles.settingsSection}>
                    <Text style={styles.sectionTitle}>Settings</Text>

                    <View style={styles.settingRow}>
                        <View style={styles.settingInfo}>
                            <Text style={styles.settingLabel}>Allow Skip</Text>
                            <Text style={styles.settingDescription}>
                                Users can skip the current song
                            </Text>
                        </View>
                        <Switch
                            value={settings.skipAllowed}
                            onValueChange={(v) => updateSetting('skipAllowed', v)}
                            trackColor={{ false: '#27272a', true: colors.primaryMuted }}
                            thumbColor="#fafafa"
                        />
                    </View>

                    <View style={styles.settingRow}>
                        <View style={styles.settingInfo}>
                            <Text style={styles.settingLabel}>Democratic Skip</Text>
                            <Text style={styles.settingDescription}>
                                Require votes to skip
                            </Text>
                        </View>
                        <Switch
                            value={settings.democraticSkip}
                            onValueChange={(v) => updateSetting('democraticSkip', v)}
                            trackColor={{ false: '#27272a', true: colors.primaryMuted }}
                            thumbColor="#fafafa"
                        />
                    </View>

                    <View style={styles.settingRow}>
                        <View style={styles.settingInfo}>
                            <Text style={styles.settingLabel}>Loop Queue</Text>
                            <Text style={styles.settingDescription}>
                                Restart from beginning when queue ends
                            </Text>
                        </View>
                        <Switch
                            value={settings.loopQueue}
                            onValueChange={(v) => updateSetting('loopQueue', v)}
                            trackColor={{ false: '#27272a', true: colors.primaryMuted }}
                            thumbColor="#fafafa"
                        />
                    </View>
                </View>
            </View>

            {error && (
                <View style={styles.errorContainer}>
                    <Text style={styles.errorText}>{error}</Text>
                </View>
            )}

            <Pressable
                style={[styles.createButton, (!name.trim() || isLoading) && styles.buttonDisabled]}
                onPress={handleCreate}
                disabled={!name.trim() || isLoading}
            >
                {isLoading ? (
                    <ActivityIndicator color={colors.textInverse} />
                ) : (
                    <Text style={styles.createButtonText}>Create Room</Text>
                )}
            </Pressable>
        </ScrollView>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background,
    },
    content: {
        paddingHorizontal: 24,
        paddingTop: 48,
        paddingBottom: 32,
    },
    header: {
        marginBottom: 32,
    },
    backButton: {
        color: colors.primary,
        fontSize: 16,
        marginBottom: 16,
    },
    title: {
        fontSize: 32,
        fontWeight: '700',
        color: colors.text,
    },
    form: {
        gap: 24,
    },
    field: {
        gap: 8,
    },
    label: {
        fontSize: 14,
        color: colors.textMuted,
        fontWeight: '500',
    },
    input: {
        backgroundColor: colors.surface,
        borderRadius: 8,
        paddingHorizontal: 16,
        paddingVertical: 16,
        color: colors.text,
        fontSize: 16,
    },
    settingsSection: {
        marginTop: 16,
        gap: 16,
    },
    sectionTitle: {
        fontSize: 18,
        fontWeight: '600',
        color: colors.text,
    },
    settingRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        backgroundColor: colors.surface,
        padding: 16,
        borderRadius: 8,
    },
    settingInfo: {
        flex: 1,
        marginRight: 16,
    },
    settingLabel: {
        fontSize: 16,
        color: colors.text,
        fontWeight: '500',
    },
    settingDescription: {
        fontSize: 14,
        color: colors.textMuted,
        marginTop: 2,
    },
    createButton: {
        backgroundColor: colors.primary,
        paddingVertical: 16,
        borderRadius: 12,
        alignItems: 'center',
        marginTop: 32,
    },
    createButtonText: {
        color: colors.textInverse,
        fontSize: 18,
        fontWeight: '600',
    },
    buttonDisabled: {
        opacity: 0.5,
    },
    errorContainer: {
        backgroundColor: colors.error + '20',
        borderColor: colors.error,
        borderWidth: 1,
        borderRadius: 8,
        padding: 12,
        marginTop: 16,
    },
    errorText: {
        color: colors.error,
        fontSize: 14,
    },
});
