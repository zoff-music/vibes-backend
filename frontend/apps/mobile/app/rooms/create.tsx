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

  const isServerMode = mode === 'server';
  const isHostMode = mode === 'host';

  const serverBtnClass = `flex-1 rounded-3xl border p-5 ${isServerMode ? 'border-theme-secondary bg-theme-secondary/10 shadow-lg shadow-cyan-500/20' : 'border-theme-border bg-theme-bg/30'}`;
  const serverTextClass = `font-heading text-[10px] tracking-[2px] ${isServerMode ? 'text-theme-secondary' : 'text-theme-text'}`;

  const hostBtnClass = `flex-1 rounded-3xl border p-5 ${isHostMode ? 'border-theme-primary bg-theme-primary/10 shadow-lg shadow-pink-500/20' : 'border-theme-border bg-theme-bg/30'}`;
  const hostTextClass = `font-heading text-[10px] tracking-[2px] ${isHostMode ? 'text-theme-primary' : 'text-theme-text'}`;

  const isInvalid = !name.trim() || isLoading;
  const opacityClass = isInvalid ? 'opacity-50' : '';
  const launchBtnClass = `h-20 flex-row items-center justify-center rounded-[32px] bg-theme-primary shadow-2xl shadow-pink-500/40 ${opacityClass}`;
  const launchBtnText = isLoading ? 'CONFIGURING...' : 'LAUNCH SESSION';

  const rocketIcon = !isLoading && (
    <FontAwesome name="rocket" size={16} color="white" />
  );

  return (
    <ScreenLayout>
      <ScrollView
        className="flex-1"
        contentContainerStyle={{ paddingHorizontal: 24, paddingBottom: 60 }}
        showsVerticalScrollIndicator={false}
      >
        <View className="py-8">
          {/* Header */}
          <View className="mb-12 flex-row items-center justify-between">
            <TouchableOpacity
              onPress={() => router.back()}
              className="p-2"
            >
              <FontAwesome name="chevron-left" size={16} color="#bfaed8" />
            </TouchableOpacity>
            <Text className="font-heading text-theme-muted text-[10px] tracking-[4px]">
              CREATE SESSION
            </Text>
          </View>

          {/* Title Area */}
          <View className="mb-12">
            <Text className="font-heading text-4xl text-theme-text leading-tight">
              LAUNCH NEW{"\n"}
              <Text className="text-theme-primary">VIBE ROOM</Text>
            </Text>
            <Text className="mt-4 font-body text-sm text-theme-text-muted">
              Configure your neon listening space.
            </Text>
          </View>

          {/* Form Content */}
          <View className="gap-8">
            {/* Primary Details */}
            <View className="rounded-[40px] border border-theme-border bg-theme-panel p-8 shadow-xl">
              <GlassInput
                label="SESSION NAME"
                placeholder="Friday Night Vibes"
                value={name}
                onChangeText={setName}
                autoFocus
                className="bg-theme-bg/30"
              />

              <View className="mt-6">
                <Text className="mb-4 font-heading text-theme-muted text-[10px] tracking-[3px]">
                  ROOM MODE
                </Text>
                <View className="flex-row gap-4">
                  <TouchableOpacity
                    onPress={() => setMode('server')}
                    className={serverBtnClass}
                  >
                    <Text className={serverTextClass}>
                      SERVER
                    </Text>
                    <Text className="mt-2 text-[10px] text-theme-text-muted leading-4">
                      Auto-play music 24/7.
                    </Text>
                  </TouchableOpacity>

                  <TouchableOpacity
                    onPress={() => setMode('host')}
                    className={hostBtnClass}
                  >
                    <Text className={hostTextClass}>
                      HOST
                    </Text>
                    <Text className="mt-2 text-[10px] text-theme-text-muted leading-4">
                      Manual control for parties.
                    </Text>
                  </TouchableOpacity>
                </View>
              </View>
            </View>

            {/* Playback Settings Group */}
            <View className="rounded-[40px] border border-theme-border bg-theme-panel p-8 shadow-xl">
              <Text className="mb-8 font-heading text-[10px] text-theme-muted tracking-[4px]">
                PLAYBACK RULES
              </Text>

              <View className="gap-2">
                <GlassSwitch
                  label="ALLOW SKIP"
                  description="Anyone can skip songs"
                  value={settings.skipAllowed}
                  onValueChange={(v) => updateSetting('skipAllowed', v)}
                />
                <GlassSwitch
                  label="DEMOCRATIC"
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
                  label="AUTO REMOVE"
                  description="Remove after play"
                  value={settings.removeOnPlay}
                  onValueChange={(v) => updateSetting('removeOnPlay', v)}
                />
              </View>
            </View>

            {/* Launch Button */}
            <TouchableOpacity
              onPress={handleCreate}
              disabled={isInvalid}
              className={launchBtnClass}
            >
              <Text className="mr-3 font-heading text-white text-sm uppercase tracking-[3px]">
                {launchBtnText}
              </Text>
              {rocketIcon}
            </TouchableOpacity>
          </View>

          <View className="mt-12 items-center">
            <Text className="font-japanese text-theme-subtle text-xs tracking-[4px] opacity-40">
              新しいセッションをつくる
            </Text>
          </View>
        </View>
      </ScrollView>
    </ScreenLayout>
  );
}
