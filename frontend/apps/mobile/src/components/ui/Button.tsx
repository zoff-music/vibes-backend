import React from 'react';
import { TouchableOpacity, StyleSheet, ActivityIndicator, ViewStyle } from 'react-native';
import { Text } from './Text';

interface Props {
    title: string;
    onPress: () => void;
    variant?: 'primary' | 'secondary' | 'outline' | 'ghost';
    size?: 'sm' | 'md' | 'lg';
    loading?: boolean;
    disabled?: boolean;
    style?: ViewStyle;
}

const THEME = {
    colors: {
        primary: '#a855f7',
        secondary: '#ec4899',
        surface: '#141414',
        text: '#fafafa',
    },
    radii: {
        md: 8,
        full: 9999,
    }
};

export const Button: React.FC<Props> = ({
    title,
    onPress,
    variant = 'primary',
    size = 'md',
    loading = false,
    disabled = false,
    style,
}) => {
    const containerStyle = [
        styles.base,
        styles[variant],
        styles[size],
        disabled && styles.disabled,
        style,
    ];

    return (
        <TouchableOpacity
            onPress={onPress}
            disabled={disabled || loading}
            style={containerStyle}
            activeOpacity={0.7}
        >
            {loading ? (
                <ActivityIndicator color="#fff" />
            ) : (
                <Text
                    bold
                    size={size === 'lg' ? 'lg' : 'md'}
                    style={variant === 'outline' || variant === 'ghost' ? { color: THEME.colors.primary } : {}}
                >
                    {title}
                </Text>
            )}
        </TouchableOpacity>
    );
};

const styles = StyleSheet.create({
    base: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        borderRadius: THEME.radii.md,
    },
    primary: {
        backgroundColor: THEME.colors.primary,
    },
    secondary: {
        backgroundColor: THEME.colors.secondary,
    },
    outline: {
        backgroundColor: 'transparent',
        borderWidth: 1,
        borderColor: THEME.colors.primary,
    },
    ghost: {
        backgroundColor: 'transparent',
    },
    sm: {
        paddingVertical: 6,
        paddingHorizontal: 12,
    },
    md: {
        paddingVertical: 10,
        paddingHorizontal: 20,
    },
    lg: {
        paddingVertical: 14,
        paddingHorizontal: 28,
    },
    disabled: {
        opacity: 0.5,
    },
});
