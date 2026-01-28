import { Stack } from 'expo-router';

export default function RoomsLayout() {
  return (
    <Stack>
      <Stack.Screen name="create" options={{ title: 'Create Room' }} />
      <Stack.Screen
        name="[id]"
        options={{ title: 'Room', headerShown: false }}
      />
      <Stack.Screen
        name="[id]/settings"
        options={{
          presentation: 'modal',
          headerShown: false,
          title: 'Settings',
        }}
      />
    </Stack>
  );
}
