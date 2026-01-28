import FontAwesome from '@expo/vector-icons/FontAwesome';
import { useRoom } from '@vibez/shared';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { ScrollView, Text, TouchableOpacity, View } from 'react-native';
import { ScreenLayout } from '../../../components/ScreenLayout';
import { GlassSwitch } from '../../../components/ui/GlassSwitch';
import { api } from '@vibez/api';

export default function RoomSettings() {
  const { id } = useLocalSearchParams();
  const roomId = typeof id === 'string' ? id : (id?.[0] ?? '');
  const router = useRouter();
  const { room, fetchRoom } = useRoom(roomId);

  const handleUpdateSetting = async (key: string, value: boolean | number) => {
    if (!room) return;
    
    // Optimistic update could happen here, but for now we rely on re-fetch/SSE
    await api.patch('/rooms/{id}/settings', { id: roomId }, {
        [key]: value
    });
    fetchRoom();
  };

  return (
    <ScreenLayout>
      <View className="flex-1">
        <View className="flex-row items-center justify-between px-6 py-4 z-10">
          <Text className="font-heading text-xs text-theme-muted tracking-[3px] uppercase">
            SETTINGS
          </Text>
          <TouchableOpacity onPress={() => router.back()} className="p-2">
            <FontAwesome name="close" size={20} color="#bfaed8" />
          </TouchableOpacity>
        </View>

        <ScrollView className="flex-1 px-6 pt-4">
             {/* General Settings */}
             <View className="bg-theme-panel p-6 rounded-3xl border border-theme-border mb-6">
                 <Text className="font-heading text-[11px] text-theme-muted tracking-[4px] mb-6">PLAYBACK</Text>
                 
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
            <View className="bg-theme-panel p-6 rounded-3xl border border-theme-border mb-10">
                <Text className="font-heading text-[11px] text-theme-muted tracking-[4px] mb-4">INFO</Text>
                <View className="flex-row justify-between mb-2">
                    <Text className="text-theme-text text-sm font-body">Room ID</Text>
                    <Text className="text-theme-muted text-sm font-mono">{roomId}</Text>
                </View>
                <View className="flex-row justify-between">
                    <Text className="text-theme-text text-sm font-body">Mode</Text>
                    <Text className="text-theme-muted text-sm uppercase font-heading tracking-widest">{room?.mode}</Text>
                </View>
            </View>
        </ScrollView>
      </View>
    </ScreenLayout>
  );
}
