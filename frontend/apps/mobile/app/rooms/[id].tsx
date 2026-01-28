import { useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { SafeAreaView, Text, View } from 'react-native';
import { Player } from '../../components/Player';

export default function RoomView() {
  const { id } = useLocalSearchParams();
  // Mock state for now
  const [videoId, _setVideoId] = useState('dQw4w9WgXcQ');
  const [playing, setPlaying] = useState(false);

  return (
    <SafeAreaView className="flex-1 bg-background">
      <View className="p-4">
        <Text className="mb-4 font-bold text-2xl text-foreground">
          Room: {id}
        </Text>

        <Player
          videoId={videoId}
          playing={playing}
          onChangeState={(state) => console.log('Player state:', state)}
        />

        <View className="mt-4 flex-row justify-center space-x-4">
          {/* Play/Pause controls mockup */}
          <Text
            className="p-2 font-bold text-primary"
            onPress={() => setPlaying(!playing)}
          >
            {playing ? 'Pause' : 'Play'}
          </Text>
        </View>

        <View className="mt-8">
          <Text className="font-bold text-foreground text-lg">Queue</Text>
          <Text className="mt-2 text-muted-foreground">No songs in queue</Text>
        </View>
      </View>
    </SafeAreaView>
  );
}
