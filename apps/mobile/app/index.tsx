import { View, Text, Pressable, TextInput } from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { createStyleSheet, useStyles } from 'react-native-unistyles';

export default function Home() {
  const { styles } = useStyles(stylesheet);
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

const stylesheet = createStyleSheet((theme) => ({
  container: {
    flex: 1,
    backgroundColor: theme.colors.background,
    paddingHorizontal: theme.spacing.lg,
    justifyContent: 'center',
  },
  header: {
    alignItems: 'center',
    marginBottom: theme.spacing.xxl,
  },
  title: {
    fontSize: theme.fontSizes.xxxl,
    fontWeight: '700',
    color: theme.colors.primary,
    marginBottom: theme.spacing.sm,
  },
  subtitle: {
    fontSize: theme.fontSizes.md,
    color: theme.colors.textMuted,
  },
  actions: {
    gap: theme.spacing.lg,
  },
  primaryButton: {
    backgroundColor: theme.colors.primary,
    paddingVertical: theme.spacing.md,
    borderRadius: theme.radii.lg,
    alignItems: 'center',
  },
  primaryButtonText: {
    color: theme.colors.textInverse,
    fontSize: theme.fontSizes.lg,
    fontWeight: '600',
  },
  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing.md,
  },
  dividerLine: {
    flex: 1,
    height: 1,
    backgroundColor: theme.colors.surfaceElevated,
  },
  dividerText: {
    color: theme.colors.textMuted,
    fontSize: theme.fontSizes.sm,
  },
  joinSection: {
    flexDirection: 'row',
    gap: theme.spacing.sm,
  },
  input: {
    flex: 1,
    backgroundColor: theme.colors.surface,
    borderRadius: theme.radii.md,
    paddingHorizontal: theme.spacing.md,
    paddingVertical: theme.spacing.md,
    color: theme.colors.text,
    fontSize: theme.fontSizes.md,
  },
  secondaryButton: {
    backgroundColor: theme.colors.surface,
    paddingHorizontal: theme.spacing.lg,
    paddingVertical: theme.spacing.md,
    borderRadius: theme.radii.md,
    justifyContent: 'center',
  },
  secondaryButtonText: {
    color: theme.colors.text,
    fontSize: theme.fontSizes.md,
    fontWeight: '500',
  },
  buttonDisabled: {
    opacity: 0.5,
  },
}));
