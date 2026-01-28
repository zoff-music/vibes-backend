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
  ...props
}: GlassButtonProps) {
  const baseStyles =
    'flex-row items-center justify-center rounded-2xl px-6 py-4 shadow-lg';

  const variants = {
    primary: 'bg-theme-primary shadow-theme-shadow-accent',
    secondary: 'bg-theme-panel border border-theme-border',
    ghost: 'bg-transparent',
  };

  const textVariants = {
    primary: 'text-white font-heading text-sm tracking-wider',
    secondary: 'text-theme-text font-heading text-sm tracking-wider',
    ghost: 'text-theme-text-muted text-sm',
  };

  return (
    <TouchableOpacity
      className={`${baseStyles} ${variants[variant]} ${className} ${props.disabled ? 'opacity-50' : ''}`}
      activeOpacity={0.8}
      {...props}
    >
      {loading ? (
        <ActivityIndicator
          color={variant === 'primary' ? 'white' : '#ff2e97'}
        />
      ) : (
        <Text className={textVariants[variant]}>{title}</Text>
      )}
    </TouchableOpacity>
  );
}
