import FontAwesome from '@expo/vector-icons/FontAwesome';
import { usePlayback, useQueue, useRoom } from '@vibez/shared';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { useEffect } from 'react';
import { ScrollView, Text, TouchableOpacity, View } from 'react-native';
import { Player } from '../../components/Player';
import { SafeCastButton } from '../../components/SafeCastButton';
import { ScreenLayout } from '../../components/ScreenLayout';

export default function RoomView() {
  const { id } = useLocalSearchParams();
  const roomId = typeof id === 'string' ? id : (id?.[0] ?? '');
  const router = useRouter();

  const { room, isLoading: isRoomLoading, fetchRoom } = useRoom(roomId);
  const { currentSong, isPlaying, play, pause, skip } = usePlayback(roomId);
  const { songs, fetchQueue } = useQueue(roomId);

  useEffect(() => {
    if (!roomId) return;

    fetchRoom();
    fetchQueue();
  }, [roomId]);

  const handleBack = () => {
    router.replace('/');
  };

  if (isRoomLoading && !room) {
    return (
      <ScreenLayout>
        <View className="flex-1 items-center justify-center">
          <Text className="font-heading text-theme-muted tracking-widest">
            CONNECTING...
          </Text>
        </View>
      </ScreenLayout>
    );
  }

  return (
    <ScreenLayout>
      <View className="flex-1">
        {/* Header */}
        <View className="z-10 flex-row items-center justify-between px-6 py-4">
          <TouchableOpacity onPress={handleBack} className="p-2">
            <FontAwesome name="chevron-left" size={16} color="#bfaed8" />
          </TouchableOpacity>

          <Text className="font-heading text-theme-muted text-xs uppercase tracking-[3px]">
            {room?.name || 'ROOM'}
          </Text>

          <View className="flex-row space-x-4">
            <SafeCastButton
              style={{ width: 24, height: 24, tintColor: '#bfaed8' }}
            />
            <TouchableOpacity 
              className="p-1"
              onPress={() => router.push(`/rooms/${roomId}/settings`)}
            >
              <FontAwesome name="cog" size={20} color="#bfaed8" />
            </TouchableOpacity>
          </View>
        </View>

        <ScrollView className="flex-1 px-6">
          {/* Player Container - CRT Frame */}
          <View className="mb-8">
            <View
              style={{
                backgroundColor: '#000',
                borderRadius: 24,
                borderWidth: 1,
                borderColor: 'rgba(0, 217, 255, 0.3)',
                overflow: 'hidden',
                aspectRatio: 16 / 9,
                width: '100%',
                shadowColor: '#00d9ff',
                shadowOffset: { width: 0, height: 0 },
                shadowOpacity: 0.3,
                shadowRadius: 20,
              }}
            >
              <Player
                videoId={currentSong?.sourceId || ''}
                playing={isPlaying}
                onChangeState={() => {}}
              />
            </View>

            {/* Now Playing Info */}
            <View className="mt-6 items-center">
              <Text
                className="mb-1 text-center font-heading text-lg text-white"
                numberOfLines={1}
              >
                {currentSong?.title || 'No Song Playing'}
              </Text>
              <Text className="text-center font-body text-theme-muted text-xs">
                {currentSong?.artist || 'Queue a song to start'}
              </Text>
            </View>

            {/* Controls */}
            <View className="mt-6 flex-row items-center justify-center space-x-8">
              <TouchableOpacity
                className="p-4"
                onPress={() => (isPlaying ? pause() : play())}
              >
                <FontAwesome
                  name={isPlaying ? 'pause' : 'play'}
                  size={24}
                  color="#bfaed8"
                />
              </TouchableOpacity>

              <TouchableOpacity className="p-4" onPress={() => skip()}>
                <FontAwesome name="step-forward" size={20} color="#bfaed8" />
              </TouchableOpacity>
            </View>
          </View>

          {/* Queue List - Glass Panel */}
          <View className="mb-8 flex-1 rounded-[32px] border border-[#2a2a30] bg-[#1a1a1e]/80 p-6">
            <View className="mb-6 flex-row items-center justify-between">
              <Text className="font-heading text-[10px] text-theme-muted tracking-[3px]">
                UP NEXT
              </Text>
              <Text className="text-[10px] text-theme-subtle">
                {songs.length} SONGS
              </Text>
            </View>

            {songs.length === 0 && (
              <View className="items-center py-8">
                <Text className="text-theme-subtle text-xs">
                  Queue is empty in this retro dimension.
                </Text>
              </View>
            )}

            {songs.length > 0 &&
              songs.map((song, index) => (
                <View
                  key={song.id}
                  className="mb-4 flex-row items-center last:mb-0"
                >
                  <View className="mr-3 h-8 w-8 items-center justify-center rounded border border-theme-border bg-theme-surface">
                    <Text className="font-heading text-theme-muted text-xs">
                      {index + 1}
                    </Text>
                  </View>
                  <View className="flex-1">
                    <Text
                      className="font-body text-sm text-white"
                      numberOfLines={1}
                    >
                      {song.title}
                    </Text>
                    <Text
                      className="text-theme-subtle text-xs"
                      numberOfLines={1}
                    >
                      {song.artist || 'Unknown Artist'}
                    </Text>
                  </View>
                </View>
              ))}
          </View>
        </ScrollView>
      </View>
    </ScreenLayout>
  );
}
