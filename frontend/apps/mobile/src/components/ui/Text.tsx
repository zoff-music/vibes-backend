import React from 'react';
import { Text as RNText, TextProps, StyleSheet } from 'react-native';

interface Props extends TextProps {
    variant?: 'body' | 'heading' | 'caption' | 'mono';
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl';
    color?: 'primary' | 'secondary' | 'muted' | 'error' | 'white';
    bold?: boolean;
}

const THEME = {
    colors: {
        text: '#fafafa',
        textMuted: '#a1a1aa',
        primary: '#a855f7',
        secondary: '#ec4899',
        error: '#ef4444',
    },
    fontSizes: {
        xs: 12,
        sm: 14,
        md: 16,
        lg: 18,
        xl: 24,
        xxl: 32,
    }
};

export const Text: React.FC<Props> = ({
    variant = 'body',
    size = 'md',
    color = 'white',
    bold = false,
    style,
    children,
    ...props
}) => {
    const textStyle = [
        styles.base,
        styles[variant],
        { fontSize: THEME.fontSizes[size] },
        color === 'muted' && { color: THEME.colors.textMuted },
        color === 'primary' && { color: THEME.colors.primary },
        color === 'secondary' && { color: THEME.colors.secondary },
        color === 'error' && { color: THEME.colors.error },
        bold && styles.bold,
        style,
    ];

    return (
        <RNText style={textStyle} {...props}>
            {children}
        </RNText>
    );
};

const styles = StyleSheet.create({
    base: {
        color: '#fafafa',
        fontFamily: 'System', // Use default for now, Inter/JetBrains later
    },
    body: {
        fontWeight: '400',
    },
    heading: {
        fontWeight: '700',
    },
    caption: {
        fontWeight: '300',
        opacity: 0.8,
    },
    mono: {
        fontFamily: 'Courier', // Placeholder for JetBrains Mono
    },
    bold: {
        fontWeight: 'bold',
    },
});
