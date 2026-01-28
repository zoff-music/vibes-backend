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

  return (
    <ScreenLayout>
      <View className="flex-1">
        <View className="z-10 flex-row items-center justify-between px-6 py-4">
          <Text className="font-heading text-theme-muted text-xs uppercase tracking-[3px]">
            SETTINGS
          </Text>
          <TouchableOpacity onPress={() => router.back()} className="p-2">
            <FontAwesome name="close" size={20} color="#bfaed8" />
          </TouchableOpacity>
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
              value={room?.settings.skipAllowed ?? true}
              onValueChange={(v) => handleUpdateSetting('skipAllowed', v)}
            />
            <GlassSwitch
              label="DEMOCRATIC SKIP"
              description="Require votes to skip"
              value={room?.settings.democraticSkip ?? false}
              onValueChange={(v) => handleUpdateSetting('democraticSkip', v)}
            />
            <GlassSwitch
              label="LOOP QUEUE"
              description="Restart when queue ends"
              value={room?.settings.loopQueue ?? false}
              onValueChange={(v) => handleUpdateSetting('loopQueue', v)}
            />
            <GlassSwitch
              label="REMOVE PLAYED"
              description="Removed after play"
              value={room?.settings.removeOnPlay ?? true}
              onValueChange={(v) => handleUpdateSetting('removeOnPlay', v)}
            />
            <GlassSwitch
              label="ALLOW DUPLICATES"
              description="Same song multiple times"
              value={room?.settings.allowDuplicates ?? false}
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
                {room?.mode}
              </Text>
            </View>
          </View>
        </ScrollView>
      </View>
    </ScreenLayout>
  );
}
