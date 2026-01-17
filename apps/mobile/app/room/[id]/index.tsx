import { View, Text } from 'react-native';
import { useLocalSearchParams } from 'expo-router';
import { createStyleSheet, useStyles } from 'react-native-unistyles';

export default function RoomView() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { styles } = useStyles(stylesheet);

  return (
    <View style={styles.container}>
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

const stylesheet = createStyleSheet((theme) => ({
  container: {
    flex: 1,
    backgroundColor: theme.colors.background,
    paddingHorizontal: theme.spacing.lg,
    paddingTop: theme.spacing.xxl,
  },
  nowPlaying: {
    alignItems: 'center',
    marginBottom: theme.spacing.xl,
  },
  artwork: {
    width: 200,
    height: 200,
    backgroundColor: theme.colors.surface,
    borderRadius: theme.radii.lg,
    marginBottom: theme.spacing.lg,
  },
  title: {
    fontSize: theme.fontSizes.xl,
    fontWeight: '600',
    color: theme.colors.text,
    marginBottom: theme.spacing.xs,
  },
  artist: {
    fontSize: theme.fontSizes.md,
    color: theme.colors.textMuted,
  },
  controls: {
    alignItems: 'center',
    paddingVertical: theme.spacing.lg,
    borderTopWidth: 1,
    borderBottomWidth: 1,
    borderColor: theme.colors.surfaceElevated,
    marginBottom: theme.spacing.lg,
  },
  controlsPlaceholder: {
    color: theme.colors.textMuted,
    fontSize: theme.fontSizes.sm,
  },
  queue: {
    flex: 1,
  },
  queueTitle: {
    fontSize: theme.fontSizes.lg,
    fontWeight: '600',
    color: theme.colors.text,
    marginBottom: theme.spacing.md,
  },
  emptyQueue: {
    color: theme.colors.textMuted,
    fontSize: theme.fontSizes.md,
    textAlign: 'center',
    paddingVertical: theme.spacing.xl,
  },
  roomId: {
    color: theme.colors.textMuted,
    fontSize: theme.fontSizes.xs,
    textAlign: 'center',
    paddingVertical: theme.spacing.md,
  },
}));
