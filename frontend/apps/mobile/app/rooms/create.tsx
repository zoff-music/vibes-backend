import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Text, TextInput, TouchableOpacity, View } from 'react-native';

export default function CreateRoom() {
  const router = useRouter();
  const [roomName, setRoomName] = useState('');

  const handleCreate = () => {
    // Logic to create room would go here
    // For now, mockup:
    const mockId = Math.random().toString(36).substring(7);
    router.push(`/rooms/${mockId}`);
  };

  return (
    <View className="flex-1 bg-background p-4">
      <Text className="mb-4 font-bold text-foreground text-xl">
        Create a new Vibe Room
      </Text>

      <TextInput
        className="mb-4 rounded-lg border border-border bg-secondary p-3 text-foreground"
        placeholder="Room Name"
        value={roomName}
        onChangeText={setRoomName}
        placeholderTextColor="#888"
      />

      <TouchableOpacity
        className="items-center rounded-lg bg-primary p-4"
        onPress={handleCreate}
      >
        <Text className="font-bold text-primary-foreground">Start Vibe</Text>
      </TouchableOpacity>
    </View>
  );
}
