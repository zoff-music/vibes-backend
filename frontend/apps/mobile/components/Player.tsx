import { useCallback, useState } from 'react';
import { ActivityIndicator, Text, View } from 'react-native';
import YoutubePlayer from 'react-native-youtube-iframe';

interface PlayerProps {
  videoId?: string;
  playing?: boolean;
  onChangeState?: (state: string) => void;
}

export function Player({
  videoId,
  playing = false,
  onChangeState,
}: PlayerProps) {
  const [ready, setReady] = useState(false);

  const onStateChange = useCallback(
    (state: string) => {
      if (state === 'ended') {
        setReady(false);
      }
      onChangeState?.(state);
    },
    [onChangeState],
  );

  if (!videoId) {
    return (
      <View className="h-56 w-full items-center justify-center rounded-lg border border-border bg-black/10">
        <Text className="text-theme-text-muted">No video playing</Text>
      </View>
    );
  }

  return (
    <View className="aspect-video w-full overflow-hidden rounded-lg bg-black">
      <YoutubePlayer
        key={videoId}
        height={220}
        play={playing} // Keep prop as fallback/initial state
        videoId={videoId}
        onChangeState={onStateChange}
        onReady={() => setReady(true)}
      />
      {!ready && (
        <View className="absolute inset-0 items-center justify-center bg-black">
          <ActivityIndicator color="#fff" />
        </View>
      )}
    </View>
  );
}
