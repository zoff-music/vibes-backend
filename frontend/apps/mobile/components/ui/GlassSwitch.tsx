import { Switch, Text, View } from 'react-native';

interface GlassSwitchProps {
  label: string;
  description?: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
}

export function GlassSwitch({
  label,
  description,
  value,
  onValueChange,
}: GlassSwitchProps) {
  return (
    <View className="mb-4 flex-row items-center justify-between rounded-2xl border border-theme-border bg-theme-panel p-5 shadow-sm">
      <View className="mr-4 flex-1">
        <Text className="mb-1 font-heading text-theme-text text-xs uppercase tracking-[2px]">
          {label}
        </Text>
        {description && (
          <Text className="font-body text-theme-text-muted text-xs">
            {description}
          </Text>
        )}
      </View>
      <Switch
        value={value}
        onValueChange={onValueChange}
        trackColor={{ false: '#2a2a30', true: '#00d9ff' }}
        thumbColor={value ? '#ffffff' : '#f4f3f4'}
      />
    </View>
  );
}
