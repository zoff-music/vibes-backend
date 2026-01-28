import { Text, TextInput, TextInputProps, View } from 'react-native';

interface GlassInputProps extends TextInputProps {
  label?: string;
  subLabel?: string;
  error?: string;
}

export function GlassInput({
  label,
  subLabel,
  error,
  className,
  ...props
}: GlassInputProps) {
  return (
    <View className="mb-4">
      {label && (
        <Text className="mb-3 font-heading text-[10px] text-theme-muted uppercase tracking-[3px]">
          {label}
          {subLabel && (
            <Text className="ml-2 text-theme-subtle normal-case tracking-normal">
              {' '}
              {subLabel}
            </Text>
          )}
        </Text>
      )}
      <TextInput
        className={`w-full rounded-2xl border border-theme-border bg-theme-panel p-4 text-base text-theme-text placeholder:text-theme-subtle ${className}`}
        placeholderTextColor="#737373"
        {...props}
      />
      {error && (
        <View className="mt-2 rounded-2xl border border-error/40 bg-error/10 p-3">
          <Text className="text-error text-sm">{error}</Text>
        </View>
      )}
    </View>
  );
}
