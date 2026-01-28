import { Switch, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function SettingsScreen() {
  return (
    <SafeAreaView className="flex-1 bg-background p-4">
      <Text className="mb-6 font-bold text-2xl text-foreground">Settings</Text>

      <View className="space-y-4">
        <View className="flex-row items-center justify-between rounded-lg border border-border bg-card p-4">
          <Text className="text-base text-foreground">Dark Mode</Text>
          <Switch value={true} />
        </View>

        <View className="rounded-lg border border-border bg-card p-4">
          <Text className="mb-1 text-base text-foreground">Version</Text>
          <Text className="text-muted-foreground text-sm">v1.0.0 (Expo)</Text>
        </View>
      </View>
    </SafeAreaView>
  );
}
