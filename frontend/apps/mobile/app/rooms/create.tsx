import FontAwesome from '@expo/vector-icons/FontAwesome';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { ScrollView, Text, TouchableOpacity, View } from 'react-native';
import { ScreenLayout } from '../../components/ScreenLayout';
import { GlassButton } from '../../components/ui/GlassButton';
import { GlassInput } from '../../components/ui/GlassInput';
import { GlassSwitch } from '../../components/ui/GlassSwitch';

const DEFAULT_SETTINGS = {
  skipAllowed: true,
  democraticSkip: true,
  loopQueue: false,
  removeOnPlay: true,
  allowDuplicates: false,
};

export default function CreateRoom() {
  const router = useRouter();

  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [mode, setMode] = useState<'server' | 'host'>('server');
  const [settings, setSettings] = useState(DEFAULT_SETTINGS);
  const [isLoading, setIsLoading] = useState(false);

  // Helper to update a single setting
  const updateSetting = (
    key: keyof typeof DEFAULT_SETTINGS,
    value: boolean,
  ) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
  };

  const handleCreate = async () => {
    if (!name.trim()) return;

    setIsLoading(true);
    // TODO: Connect to real API
    // Simulate API delay
    setTimeout(() => {
      setIsLoading(false);
      // Replace with actual API call
      const roomId = Math.random().toString(36).substring(7);
      router.replace(`/rooms/${roomId}`);
    }, 1000);
  };

  return (
    <ScreenLayout>
      <ScrollView className="flex-1 px-6">
        <View className="py-8">
          {/* Header */}
          <View className="mb-8 flex-row items-center justify-between">
            <TouchableOpacity
              onPress={() => router.back()}
              className="flex-row items-center"
            >
              <FontAwesome name="arrow-left" size={16} color="#bfaed8" />
              <Text className="ml-2 font-heading text-theme-muted text-xs tracking-[3px]">
                BACK
              </Text>
            </TouchableOpacity>
            <Text className="font-heading text-theme-muted text-xs tracking-[3px]">
              CREATE SESSION
            </Text>
          </View>

          {/* Title */}
          <View className="mb-10 items-center">
            <Text className="mb-2 text-center font-heading text-4xl text-theme-text">
              CREATE A SESSION
            </Text>
            <Text className="font-body text-sm text-theme-muted">
              Build a neon listening room in seconds.
            </Text>
            <Text className="mt-2 font-japanese text-theme-subtle text-xs tracking-widest">
              セッションを作成
            </Text>
          </View>

          {/* Form Content */}
          <View className="space-y-6">
            {/* Name & Mode Section */}
            <View className="space-y-6">
              <View className="rounded-3xl border border-theme-border bg-theme-panel p-6">
                <GlassInput
                  label="SESSION NAME"
                  placeholder="Friday Night Vibes"
                  value={name}
                  onChangeText={setName}
                  autoFocus
                />
              </View>

              <View className="rounded-3xl border border-theme-border bg-theme-panel p-6">
                <Text className="mb-4 font-heading text-theme-muted text-xs tracking-[3px]">
                  ROOM MODE
                </Text>
                <View className="flex-row space-x-4">
                  <TouchableOpacity
                    onPress={() => setMode('server')}
                    className={`flex-1 rounded-2xl border p-4 ${mode === 'server' ? 'border-theme-secondary bg-secondary/10 shadow-cyan-500/20 shadow-lg' : 'border-theme-border bg-theme-surface'}`}
                  >
                    <Text className="mb-2 font-heading text-[10px] text-theme-text tracking-[2px]">
                      SERVER MODE
                    </Text>
                    <Text className="text-[10px] text-theme-muted">
                      Auto-play music 24/7 for radio rooms.
                    </Text>
                  </TouchableOpacity>

                  <TouchableOpacity
                    onPress={() => setMode('host')}
                    className={`flex-1 rounded-2xl border p-4 ${mode === 'host' ? 'border-theme-primary bg-primary/10 shadow-lg shadow-pink-500/20' : 'border-theme-border bg-theme-surface'}`}
                  >
                    <Text className="mb-2 font-heading text-[10px] text-theme-text tracking-[2px]">
                      HOST MODE
                    </Text>
                    <Text className="text-[10px] text-theme-muted">
                      Host controls playback for parties.
                    </Text>
                  </TouchableOpacity>
                </View>
              </View>

              <View className="rounded-3xl border border-theme-border bg-theme-panel p-6">
                <GlassInput
                  label="ADMIN PASSWORD"
                  subLabel="(optional)"
                  placeholder="For room control"
                  secureTextEntry
                  value={password}
                  onChangeText={setPassword}
                />
                <Text className="mt-2 text-theme-subtle text-xs">
                  Leave empty to allow anyone to control playback.
                </Text>
              </View>
            </View>

            {/* Settings Section */}
            <View className="rounded-3xl border border-theme-border bg-theme-panel p-6">
              <Text className="mb-6 font-heading text-[11px] text-theme-muted tracking-[4px]">
                PLAYBACK SETTINGS
              </Text>

              <GlassSwitch
                label="ALLOW SKIP"
                description="Anyone can skip songs"
                value={settings.skipAllowed}
                onValueChange={(v) => updateSetting('skipAllowed', v)}
              />
              <GlassSwitch
                label="DEMOCRATIC SKIP"
                description="Require votes to skip"
                value={settings.democraticSkip}
                onValueChange={(v) => updateSetting('democraticSkip', v)}
              />
              <GlassSwitch
                label="LOOP QUEUE"
                description="Restart when queue ends"
                value={settings.loopQueue}
                onValueChange={(v) => updateSetting('loopQueue', v)}
              />
              <GlassSwitch
                label="REMOVE PLAYED"
                description="Removed after play"
                value={settings.removeOnPlay}
                onValueChange={(v) => updateSetting('removeOnPlay', v)}
              />
              <GlassSwitch
                label="ALLOW DUPLICATES"
                description="Same song multiple times"
                value={settings.allowDuplicates}
                onValueChange={(v) => updateSetting('allowDuplicates', v)}
              />
            </View>

            {/* Submit Button */}
            <View className="pt-4 pb-10">
              <GlassButton
                title="START SESSION"
                onPress={handleCreate}
                disabled={!name.trim() || isLoading}
                loading={isLoading}
              />
            </View>
          </View>
        </View>
      </ScrollView>
    </ScreenLayout>
  );
}
