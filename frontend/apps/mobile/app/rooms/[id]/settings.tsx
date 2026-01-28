import FontAwesome from '@expo/vector-icons/FontAwesome';
import { api } from '@vibez/api';
import { useRoom } from '@vibez/shared';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { ScrollView, Text, TouchableOpacity, View } from 'react-native';
import { ScreenLayout } from '../../../components/ScreenLayout';
import { GlassSwitch } from '../../../components/ui/GlassSwitch';

export default function RoomSettings() {
  const { id } = useLocalSearchParams();
  const roomId = typeof id === 'string' ? id : (id?.[0] ?? '');
  const router = useRouter();
  const { room, fetchRoom } = useRoom(roomId);

  const handleUpdateSetting = async (key: string, value: boolean | number) => {
    if (!room) return;

    // Optimistic update could happen here, but for now we rely on re-fetch/SSE
    await api.patch(
      '/rooms/{id}/settings',
      { id: roomId },
      {
        [key]: value,
      },
    );
    fetchRoom();
  };

  const headerTitle = room?.name || 'BACK';
  const skipAllowedValue = room?.settings.skipAllowed ?? true;
  const democraticSkipValue = room?.settings.democraticSkip ?? false;
  const loopQueueValue = room?.settings.loopQueue ?? false;
  const removeOnPlayValue = room?.settings.removeOnPlay ?? true;
  const allowDuplicatesValue = room?.settings.allowDuplicates ?? false;
  const roomModeText = room?.mode;

  return (
    <ScreenLayout>
      <View className="flex-1">
        <View className="z-10 flex-row items-center justify-between px-6 py-4">
          <TouchableOpacity
            onPress={() => router.back()}
            className="flex-row items-center p-2"
          >
            <FontAwesome name="chevron-left" size={16} color="#bfaed8" />
            <Text className="ml-3 font-heading text-theme-muted text-xs uppercase tracking-[2px]">
              {headerTitle}
            </Text>
          </TouchableOpacity>

          <Text className="font-heading text-theme-muted text-[10px] uppercase tracking-[3px] opacity-50">
            SETTINGS
          </Text>
        </View>

        <ScrollView className="flex-1 px-6 pt-4">
          {/* General Settings */}
          <View className="mb-6 rounded-3xl border border-theme-border bg-theme-panel p-6">
            <Text className="mb-6 font-heading text-[11px] text-theme-muted tracking-[4px]">
              PLAYBACK
            </Text>

            <GlassSwitch
              label="ALLOW SKIP"
              description="Anyone can skip songs"
              value={skipAllowedValue}
              onValueChange={(v) => handleUpdateSetting('skipAllowed', v)}
            />
            <GlassSwitch
              label="DEMOCRATIC SKIP"
              description="Require votes to skip"
              value={democraticSkipValue}
              onValueChange={(v) => handleUpdateSetting('democraticSkip', v)}
            />
            <GlassSwitch
              label="LOOP QUEUE"
              description="Restart when queue ends"
              value={loopQueueValue}
              onValueChange={(v) => handleUpdateSetting('loopQueue', v)}
            />
            <GlassSwitch
              label="REMOVE PLAYED"
              description="Removed after play"
              value={removeOnPlayValue}
              onValueChange={(v) => handleUpdateSetting('removeOnPlay', v)}
            />
            <GlassSwitch
              label="ALLOW DUPLICATES"
              description="Same song multiple times"
              value={allowDuplicatesValue}
              onValueChange={(v) => handleUpdateSetting('allowDuplicates', v)}
            />
          </View>

          {/* Admin Info */}
          <View className="mb-10 rounded-3xl border border-theme-border bg-theme-panel p-6">
            <Text className="mb-4 font-heading text-[11px] text-theme-muted tracking-[4px]">
              INFO
            </Text>
            <View className="mb-2 flex-row justify-between">
              <Text className="font-body text-sm text-theme-text">Room ID</Text>
              <Text className="font-mono text-sm text-theme-muted">
                {roomId}
              </Text>
            </View>
            <View className="flex-row justify-between">
              <Text className="font-body text-sm text-theme-text">Mode</Text>
              <Text className="font-heading text-sm text-theme-muted uppercase tracking-widest">
                {roomModeText}
              </Text>
            </View>
          </View>
        </ScrollView>
      </View>
    </ScreenLayout>
  );
}
