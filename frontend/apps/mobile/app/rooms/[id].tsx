import FontAwesome from '@expo/vector-icons/FontAwesome';
import { usePlayback, useQueue, useRoom } from '@vibez/shared';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { useEffect } from 'react';
import { Alert, ScrollView, Text, TouchableOpacity, View } from 'react-native';
import { Player } from '../../components/Player';
import { SafeCastButton } from '../../components/SafeCastButton';
import { ScreenLayout } from '../../components/ScreenLayout';

export default function RoomView() {
  const { id } = useLocalSearchParams();
  const roomId = typeof id === 'string' ? id : (id?.[0] ?? '');
  const router = useRouter();

  const { room, isLoading: isRoomLoading, fetchRoom } = useRoom(roomId);
  const { currentSong, isPlaying, play, pause, skip, fetchPlayback } =
    usePlayback(roomId, {
      onToast: (message, type) => {
        if (type === 'error') {
          Alert.alert('Error', message);
        } else if (type === 'info') {
          // Optional: Alert.alert('Info', message);
        }
      },
    });
  const { songs, fetchQueue } = useQueue(roomId);

  useEffect(() => {
    if (!roomId) return;

    fetchRoom();
    fetchQueue();
    fetchPlayback();
  }, [roomId]);

  const handleBack = () => {
    if (router.canGoBack()) {
      router.back();
    } else {
      router.replace('/');
    }
  };

  const playPauseAction = isPlaying ? pause : play;
  const playPauseIcon = isPlaying ? 'pause' : 'play';
  const playPauseColor = isPlaying ? '#ff2e97' : '#00d9ff';
  const playPauseBg = isPlaying
    ? 'rgba(255, 46, 151, 0.1)'
    : 'rgba(0, 217, 255, 0.1)';
  const playPauseBorder = isPlaying
    ? 'rgba(255, 46, 151, 0.4)'
    : 'rgba(0, 217, 255, 0.4)';
  const playPauseShadowColor = isPlaying ? '#ff2e97' : '#00d9ff';
  const playPauseMarginLeft = isPlaying ? 0 : 4;

  const currentSongVideoId = currentSong?.sourceId || '';
  const currentSongTitle = currentSong?.title || 'System Idle';
  const currentSongArtist = currentSong?.artist || 'Ready for input...';

  const emptyQueueElement = (
    <View className="items-center py-12">
      <View className="mb-4 h-12 w-12 items-center justify-center rounded-2xl bg-theme-bg/30">
        <FontAwesome name="music" size={20} color="#6b6b73" />
      </View>
      <Text className="text-center font-body text-theme-subtle text-xs leading-5">
        The signal is clear.{'\n'}Add some tracks to start the sequence.
      </Text>
    </View>
  );

  const queueListElement = (
    <View className="gap-6">
      {songs.map((song, index) => {
        const songNumber = String(index + 1).padStart(2, '0');
        const songArtist = song.artist || 'Unknown Signal';

        return (
          <View key={song.id} className="flex-row items-center">
            <View className="mr-4 h-10 w-10 items-center justify-center rounded-2xl border border-theme-border/50 bg-theme-bg/50">
              <Text className="font-heading text-theme-primary text-xs">
                {songNumber}
              </Text>
            </View>
            <View className="flex-1">
              <Text
                className="mb-1 font-heading text-[13px] text-theme-text"
                numberOfLines={1}
              >
                {song.title}
              </Text>
              <Text
                className="font-body text-[11px] text-theme-text-muted"
                numberOfLines={1}
              >
                {songArtist}
              </Text>
            </View>
          </View>
        );
      })}
    </View>
  );

  const queueContent =
    songs.length === 0 ? emptyQueueElement : queueListElement;

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

          <View className="flex-row items-center space-x-6">
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

        <ScrollView
          className="flex-1"
          contentContainerStyle={{ paddingHorizontal: 24, paddingBottom: 40 }}
          showsVerticalScrollIndicator={false}
        >
          {/* Player Container */}
          <View className="mt-4 mb-10">
            <View
              style={{
                backgroundColor: '#000',
                borderRadius: 32,
                borderWidth: 1,
                borderColor: 'rgba(0, 217, 255, 0.2)',
                overflow: 'hidden',
                aspectRatio: 16 / 9,
                width: '100%',
                shadowColor: '#00d9ff',
                shadowOffset: { width: 0, height: 20 },
                shadowOpacity: 0.2,
                shadowRadius: 30,
                elevation: 10,
              }}
            >
              <Player
                videoId={currentSongVideoId}
                playing={isPlaying}
                onChangeState={() => {}}
              />
            </View>

            {/* Now Playing Info */}
            <View className="mt-8">
              <Text className="mb-2 font-heading text-[10px] text-theme-primary uppercase tracking-[4px]">
                NOW PLAYING
              </Text>
              <Text
                className="font-heading text-2xl text-theme-text"
                numberOfLines={2}
              >
                {currentSongTitle}
              </Text>
              <Text className="mt-2 font-body text-sm text-theme-text-muted">
                {currentSongArtist}
              </Text>
            </View>

            {/* Controls */}
            <View className="mt-10 flex-row items-center justify-center gap-10">
              <TouchableOpacity
                onPress={playPauseAction}
                style={{
                  width: 80,
                  height: 80,
                  borderRadius: 40,
                  backgroundColor: playPauseBg,
                  borderWidth: 1,
                  borderColor: playPauseBorder,
                  alignItems: 'center',
                  justifyContent: 'center',
                  shadowColor: playPauseShadowColor,
                  shadowOffset: { width: 0, height: 0 },
                  shadowOpacity: 0.3,
                  shadowRadius: 15,
                }}
              >
                <FontAwesome
                  name={playPauseIcon}
                  size={32}
                  color={playPauseColor}
                  style={{ marginLeft: playPauseMarginLeft }}
                />
              </TouchableOpacity>

              <TouchableOpacity
                onPress={() => skip()}
                style={{
                  width: 60,
                  height: 60,
                  borderRadius: 30,
                  backgroundColor: 'rgba(191, 174, 216, 0.05)',
                  borderWidth: 1,
                  borderColor: 'rgba(191, 174, 216, 0.2)',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <FontAwesome name="step-forward" size={20} color="#bfaed8" />
              </TouchableOpacity>
            </View>
          </View>

          {/* Queue List - Glass Panel */}
          <View className="rounded-[40px] border border-theme-border bg-theme-panel p-8 shadow-xl">
            <View className="mb-8 flex-row items-center justify-between">
              <View>
                <Text className="font-heading text-[10px] text-theme-muted uppercase tracking-[4px]">
                  UP NEXT
                </Text>
                <Text className="mt-1 font-body text-[10px] text-theme-subtle">
                  {songs.length} TRACKS SYNCED
                </Text>
              </View>
              <View className="h-8 w-8 items-center justify-center rounded-full bg-theme-bg/50">
                <FontAwesome name="list" size={10} color="#bfaed8" />
              </View>
            </View>

            {queueContent}
          </View>
        </ScrollView>
      </View>
    </ScreenLayout>
  );
}
