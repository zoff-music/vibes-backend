import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import '../src/styles/unistyles';

export default function RootLayout() {
  return (
    <>
      <StatusBar style="light" />
      <Stack
        screenOptions={{
          headerShown: false,
          contentStyle: { backgroundColor: '#0a0a0a' },
        }}
      />
    </>
  );
}
