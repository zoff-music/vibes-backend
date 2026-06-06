import { GlassView } from 'expo-glass-effect';
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
  const thumbColor = value ? '#ffffff' : '#f4f3f4';
  const trackColor = { false: '#2a2a30', true: '#00d9ff' };
  const containerStyle = {
    padding: 20,
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    justifyContent: 'space-between' as const,
  };

  const descriptionElement = description && (
    <Text className="font-body text-theme-text-muted text-xs">
      {description}
    </Text>
  );

  return (
    <View className="mb-4 overflow-hidden rounded-2xl border border-theme-border/30">
      <GlassView
        glassEffectStyle="regular"
        tintColor="rgba(255, 255, 255, 0.05)"
        style={containerStyle}
      >
        <View className="mr-4 flex-1">
          <Text className="mb-1 font-heading text-theme-text text-xs uppercase tracking-[2px]">
            {label}
          </Text>
          {descriptionElement}
        </View>
        <Switch
          value={value}
          onValueChange={onValueChange}
          trackColor={trackColor}
          thumbColor={thumbColor}
        />
      </GlassView>
    </View>
  );
}
