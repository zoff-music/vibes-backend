import { View, Text, Pressable, TextInput, Switch } from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { createStyleSheet, useStyles } from 'react-native-unistyles';
import { DEFAULT_ROOM_SETTINGS } from '@vibez/shared';

export default function CreateRoom() {
  const { styles } = useStyles(stylesheet);
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [settings, setSettings] = useState(DEFAULT_ROOM_SETTINGS);

  const handleCreate = async () => {
    // TODO: Call API to create room
    // For now, just navigate with a placeholder
    router.replace('/room/new-room-id');
  };

  const updateSetting = <K extends keyof typeof settings>(
    key: K,
    value: (typeof settings)[K]
  ) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <View style={styles.container}>
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
              trackColor={{ false: '#27272a', true: '#7c3aed' }}
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
              trackColor={{ false: '#27272a', true: '#7c3aed' }}
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
              trackColor={{ false: '#27272a', true: '#7c3aed' }}
              thumbColor="#fafafa"
            />
          </View>
        </View>
      </View>

      <Pressable
        style={[styles.createButton, !name.trim() && styles.buttonDisabled]}
        onPress={handleCreate}
        disabled={!name.trim()}
      >
        <Text style={styles.createButtonText}>Create Room</Text>
      </Pressable>
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
  header: {
    marginBottom: theme.spacing.xl,
  },
  backButton: {
    color: theme.colors.primary,
    fontSize: theme.fontSizes.md,
    marginBottom: theme.spacing.md,
  },
  title: {
    fontSize: theme.fontSizes.xxl,
    fontWeight: '700',
    color: theme.colors.text,
  },
  form: {
    flex: 1,
    gap: theme.spacing.lg,
  },
  field: {
    gap: theme.spacing.sm,
  },
  label: {
    fontSize: theme.fontSizes.sm,
    color: theme.colors.textMuted,
    fontWeight: '500',
  },
  input: {
    backgroundColor: theme.colors.surface,
    borderRadius: theme.radii.md,
    paddingHorizontal: theme.spacing.md,
    paddingVertical: theme.spacing.md,
    color: theme.colors.text,
    fontSize: theme.fontSizes.md,
  },
  settingsSection: {
    marginTop: theme.spacing.md,
    gap: theme.spacing.md,
  },
  sectionTitle: {
    fontSize: theme.fontSizes.lg,
    fontWeight: '600',
    color: theme.colors.text,
  },
  settingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: theme.colors.surface,
    padding: theme.spacing.md,
    borderRadius: theme.radii.md,
  },
  settingInfo: {
    flex: 1,
    marginRight: theme.spacing.md,
  },
  settingLabel: {
    fontSize: theme.fontSizes.md,
    color: theme.colors.text,
    fontWeight: '500',
  },
  settingDescription: {
    fontSize: theme.fontSizes.sm,
    color: theme.colors.textMuted,
    marginTop: 2,
  },
  createButton: {
    backgroundColor: theme.colors.primary,
    paddingVertical: theme.spacing.md,
    borderRadius: theme.radii.lg,
    alignItems: 'center',
    marginBottom: theme.spacing.xl,
  },
  createButtonText: {
    color: theme.colors.textInverse,
    fontSize: theme.fontSizes.lg,
    fontWeight: '600',
  },
  buttonDisabled: {
    opacity: 0.5,
  },
}));
