import { useLocalSearchParams, useRouter } from 'expo-router';
import { useEffect } from 'react';
import { Text, View } from 'react-native';

// This handles the redirect from Spotify/OAuth
export default function Callback() {
  const router = useRouter();
  const params = useLocalSearchParams();

  useEffect(() => {
    // Log params to see what we get from the deep link
    console.log('Callback params:', params);

    // If we have code or error, handle it
    if (Object.keys(params).length > 0) {
      // Simulate auth processing
      setTimeout(() => {
        router.replace('/');
      }, 1000);
    }
  }, [params]);

  return (
    <View className="flex-1 items-center justify-center bg-background">
      <Text className="text-foreground">Authenticating...</Text>
    </View>
  );
}
