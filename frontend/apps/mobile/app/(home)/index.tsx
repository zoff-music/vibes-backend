import FontAwesome from '@expo/vector-icons/FontAwesome';
import { api } from '@vibez/api';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Text, TouchableOpacity, View } from 'react-native';
import { SafeCastButton } from '../../components/SafeCastButton';
import { ScreenLayout } from '../../components/ScreenLayout';
import { GlassInput } from '../../components/ui/GlassInput';

export default function HomeScreen() {
  const router = useRouter();
  const [roomCode, setRoomCode] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleJoin = async () => {
    if (!roomCode.trim() || isLoading) return;

    setIsLoading(true);
    const slug = roomCode.trim().toLowerCase();

    const [err] = await api.get('/rooms/{id}', { id: slug });

    setIsLoading(false);

    if (err) {
      router.push(`/rooms/create?name=${encodeURIComponent(slug)}`);
      return;
    }

    router.push(`/rooms/${slug}`);
  };

  const isInvalid = !roomCode.trim() || isLoading;
  const opacityClass = isInvalid ? 'opacity-50' : '';
  const joinBtnClass = `h-16 flex-row items-center justify-center rounded-3xl bg-theme-secondary shadow-lg shadow-cyan-500/30 ${opacityClass}`;
  const joinBtnText = isLoading ? 'SEARCHING...' : 'JOIN ROOM';

  const chevronIcon = !isLoading && (
    <FontAwesome name="chevron-right" size={12} color="white" />
  );

  return (
    <ScreenLayout>
      {/* Header Elements */}
      <View className="absolute top-14 right-6 z-20">
        <SafeCastButton
          style={{ width: 32, height: 32, tintColor: '#bfaed8' }}
        />
      </View>

      <View className="flex-1 justify-center px-6">
        <View className="mb-12 items-center">
          {/* Logo Section */}
          <View className="mb-4 items-center justify-center">
            <Text
              className="font-heading text-8xl text-theme-text"
              style={{
                textShadowColor: 'rgba(255, 46, 151, 0.6)',
                textShadowOffset: { width: 0, height: 0 },
                textShadowRadius: 30,
              }}
            >
              ノリ
            </Text>
            <View className="absolute -bottom-2">
              <Text className="font-heading text-[10px] text-theme-primary uppercase tracking-[8px]">
                VIBES
              </Text>
            </View>
          </View>

          <Text className="mt-8 text-center font-body text-sm text-theme-text-muted leading-6">
            Connect your world through sound.{'\n'}
            Join a room or start your own neon session.
          </Text>
        </View>

        {/* Action Card */}
        <View className="rounded-[40px] border border-theme-border bg-theme-panel p-8 shadow-2xl">
          <View className="mb-8">
            <Text className="mb-4 font-heading text-[10px] text-theme-muted uppercase tracking-[4px]">
              JOIN SESSION
            </Text>
            <GlassInput
              placeholder="e.g. night-city"
              value={roomCode}
              onChangeText={setRoomCode}
              autoCapitalize="none"
              onSubmitEditing={handleJoin}
              returnKeyType="join"
              autoCorrect={false}
              className="border-theme-border bg-theme-bg/50 text-center font-heading text-lg"
            />
          </View>

          <View className="gap-4">
            <TouchableOpacity
              onPress={handleJoin}
              disabled={isInvalid}
              className={joinBtnClass}
            >
              <Text className="mr-2 font-heading text-white text-xs uppercase tracking-[2px]">
                {joinBtnText}
              </Text>
              {chevronIcon}
            </TouchableOpacity>

            <TouchableOpacity
              onPress={() => router.push('/rooms/create')}
              className="h-16 items-center justify-center rounded-3xl border border-theme-primary/30 bg-theme-primary/10"
            >
              <Text className="font-heading text-theme-primary text-xs uppercase tracking-[2px]">
                CREATE NEW
              </Text>
            </TouchableOpacity>
          </View>
        </View>

        <Text className="mt-8 text-center font-japanese text-theme-subtle text-xs tracking-widest opacity-40">
          音楽の共有 🌌 ネオンの夜
        </Text>
      </View>
    </ScreenLayout>
  );
}
