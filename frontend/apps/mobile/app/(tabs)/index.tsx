import { Link, useRouter } from 'expo-router';
import { useState } from 'react';
import { Text, TextInput, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function HomeScreen() {
  const router = useRouter();
  const [joinId, setJoinId] = useState('');

  const handleJoin = () => {
    if (joinId) {
      router.push(`/rooms/${joinId}`);
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-background p-4">
      <View className="flex-1 items-center justify-center space-y-8">
        <View className="mb-8 items-center">
          <Text className="mb-2 font-bold text-4xl text-foreground">VibeZ</Text>
          <Text className="text-lg text-muted-foreground">
            Listen together.
          </Text>
        </View>

        <View className="w-full max-w-sm space-y-4">
          <View>
            <Text className="mb-2 font-medium text-foreground">
              Join a Room
            </Text>
            <View className="flex-row space-x-2">
              <TextInput
                className="flex-1 rounded-lg border border-border bg-secondary p-3 text-foreground"
                placeholder="Room ID"
                placeholderTextColor="#666"
                value={joinId}
                onChangeText={setJoinId}
                autoCapitalize="none"
              />
              <TouchableOpacity
                className="items-center justify-center rounded-lg bg-primary p-3 px-6"
                onPress={handleJoin}
              >
                <Text className="font-bold text-primary-foreground">Join</Text>
              </TouchableOpacity>
            </View>
          </View>

          <View className="items-center py-4">
            <Text className="text-muted-foreground">or</Text>
          </View>

          <Link href="/rooms/create" asChild>
            <TouchableOpacity className="w-full items-center rounded-lg border border-border bg-secondary p-4">
              <Text className="font-bold text-foreground">Create a Room</Text>
            </TouchableOpacity>
          </Link>
        </View>

        <View className="mt-8 w-full max-w-sm">
          <Text className="mb-4 font-bold text-foreground text-lg">
            Recent Vibes
          </Text>
          {/* Placeholder for recent rooms - could be fetched from local storage or API */}
          <View className="rounded-lg border border-border bg-card p-4">
            <Text className="text-center text-muted-foreground">
              No recent rooms
            </Text>
          </View>
        </View>
      </View>
    </SafeAreaView>
  );
}
