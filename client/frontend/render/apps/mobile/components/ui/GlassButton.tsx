import { GlassView } from 'expo-glass-effect';
import {
  ActivityIndicator,
  Text,
  TouchableOpacity,
  TouchableOpacityProps,
} from 'react-native';

interface GlassButtonProps extends TouchableOpacityProps {
  title: string;
  loading?: boolean;
  variant?: 'primary' | 'secondary' | 'ghost';
}

export function GlassButton({
  title,
  loading,
  variant = 'primary',
  className,
  style,
  ...props
}: GlassButtonProps) {
  const baseStyles =
    'flex-row items-center justify-center rounded-2xl overflow-hidden';

  const variants = {
    primary: 'border-white/20',
    secondary: 'border-theme-border/30',
    ghost: '',
  };

  const textVariants = {
    primary: 'text-white font-heading text-sm tracking-wider',
    secondary: 'text-theme-text font-heading text-sm tracking-wider',
    ghost: 'text-theme-text-muted text-sm',
  };

  const isDisabled = props.disabled;
  const opacityClass = isDisabled ? 'opacity-50' : '';
  const buttonClass = `${baseStyles} ${variants[variant]} ${className} ${opacityClass}`;
  const ghostButtonClass = `${baseStyles} ${className} ${opacityClass}`;

  const activityIndicatorColor = variant === 'primary' ? 'white' : '#ff2e97';
  const textClass = textVariants[variant];
  const glassEffectStyle = 'regular';
  const glassTintColor =
    variant === 'primary'
      ? 'rgba(255, 46, 151, 0.4)'
      : 'rgba(255, 255, 255, 0.1)';
  const _intensity = variant === 'primary' ? 60 : 30; // Just in case, though we used expo-glass-effect

  const content = (
    <>
      {loading && <ActivityIndicator color={activityIndicatorColor} />}
      {!loading && <Text className={textClass}>{title}</Text>}
    </>
  );

  if (variant === 'ghost') {
    return (
      <TouchableOpacity
        className={ghostButtonClass}
        activeOpacity={0.8}
        {...props}
      >
        {content}
      </TouchableOpacity>
    );
  }

  return (
    <TouchableOpacity className={buttonClass} activeOpacity={0.8} {...props}>
      <GlassView
        glassEffectStyle={glassEffectStyle}
        tintColor={glassTintColor}
        style={[
          {
            width: '100%',
            paddingVertical: 16,
            paddingHorizontal: 24,
            alignItems: 'center',
            justifyContent: 'center',
          },
          style,
        ]}
      >
        {content}
      </GlassView>
    </TouchableOpacity>
  );
}
