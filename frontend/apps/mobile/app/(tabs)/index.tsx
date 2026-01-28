import { api } from '@vibez/api';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Text, TouchableOpacity, View } from 'react-native';
import { SafeCastButton } from '../../components/SafeCastButton';
import { ScreenLayout } from '../../components/ScreenLayout';
import { GlassInput } from '../../components/ui/GlassInput';

// ...

export default function HomeScreen() {
  const router = useRouter();
  const [roomCode, setRoomCode] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  // Checks if room exists before joining
  const handleJoin = async () => {
    if (!roomCode.trim() || isLoading) return;

    setIsLoading(true);
    const slug = roomCode.trim().toLowerCase();

    // Use safeWrapAsync as requested, though api.get already returns [err, data]
    // properly wrapped. However, since the goal is to avoid try/catch blocks completely:
    // The previous code had a manual try/catch.
    // api.get is already "safe" (returns tuple).

    const [err] = await api.get('/rooms/{id}', { id: slug });

    setIsLoading(false);

    if (err) {
      // Room not found or error -> go to create (prefilled)
      router.push(`/rooms/create?name=${encodeURIComponent(slug)}`);
      return;
    }

    // Room exists -> join
    router.push(`/rooms/${slug}`);
  };

  return (
    <ScreenLayout>
      {/* Header Elements */}
      <View className="absolute top-14 right-6 z-20">
        <SafeCastButton
          style={{ width: 32, height: 32, tintColor: '#bfaed8' }}
        />
      </View>

      <View className="flex-1 items-center justify-center px-6">
        {/* CRT Frame Container */}
        <View
          style={{
            width: '100%',
            maxWidth: 400,
            backgroundColor: 'rgba(6, 3, 15, 0.65)', // crt-bg
            borderColor: 'rgba(0, 217, 255, 0.2)', // theme-border
            borderWidth: 1,
            borderRadius: 36,
            padding: 24,
            shadowColor: '#000',
            shadowOffset: { width: 0, height: 10 },
            shadowOpacity: 0.5,
            shadowRadius: 20,
          }}
        >
          <View className="mb-10 items-center">
            <Text className="mb-4 font-heading text-[#a1a1a8] text-[10px] tracking-[4px]">
              ENTER THE VIBE
            </Text>

            {/* The Japanese Logo "Nori" from Platform */}
            <Text
              className="font-heading text-6xl text-[#f7efff]"
              style={{
                textShadowColor: 'rgba(255, 46, 151, 0.8)',
                textShadowOffset: { width: 0, height: 0 },
                textShadowRadius: 20,
              }}
            >
              ノリ
            </Text>

            <Text className="mt-4 text-center font-body text-[#a1a1a8] text-sm">
              Shared music rooms for neon nights.
            </Text>
            <Text className="mt-2 text-[#6b6b73] text-xs tracking-widest">
              音楽を共有
            </Text>
          </View>

          {/* Form Section */}
          <View className="space-y-6">
            <View className="rounded-[24px] bg-[#1a1a1e] p-6">
              <Text className="mb-3 font-heading text-[#a1a1a8] text-[10px] tracking-[3px]">
                ROOM NAME
              </Text>
              <GlassInput
                placeholder="Enter Room Name..."
                value={roomCode}
                onChangeText={setRoomCode}
                autoCapitalize="none"
                className="border-[#2a2a30] bg-[#140b2b] text-center text-[#f7efff]"
              />
            </View>

            <View className="flex-row gap-4">
              <View className="flex-1">
                <TouchableOpacity
                  onPress={() => router.push('/rooms/create')}
                  className="items-center justify-center rounded-2xl border border-[#ff2e97]/50 bg-[#ff2e97]/90 py-4 shadow-[#ff2e97]/40 shadow-lg"
                >
                  <Text className="font-heading text-white text-xs uppercase">
                    Start Session
                  </Text>
                </TouchableOpacity>
              </View>

              <View className="flex-1">
                <TouchableOpacity
                  onPress={handleJoin}
                  disabled={!roomCode.trim() || isLoading}
                  className={`items-center justify-center rounded-2xl border border-[#00d9ff]/50 bg-[#00d9ff]/85 py-4 shadow-[#00d9ff]/40 shadow-lg ${!roomCode.trim() || isLoading ? 'opacity-50' : ''}`}
                >
                  <Text className="font-heading text-white text-xs uppercase">
                    {isLoading ? '...' : 'Join Room'}
                  </Text>
                </TouchableOpacity>
              </View>
            </View>
          </View>
        </View>
      </View>
    </ScreenLayout>
  );
}
