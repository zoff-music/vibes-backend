import React from 'react';
import { TextInput, StyleSheet, View, TextInputProps, ViewStyle } from 'react-native';
import { Text } from './Text';

interface Props extends TextInputProps {
    label?: string;
    error?: string;
    containerStyle?: ViewStyle;
}

export const Input: React.FC<Props> = ({ label, error, containerStyle, style, ...props }) => {
    return (
        <View style={[styles.container, containerStyle]}>
            {label && <Text size="sm" color="muted" style={styles.label}>{label}</Text>}
            <TextInput
                style={[
                    styles.input,
                    error && styles.inputError,
                    style,
                ]}
                placeholderTextColor="#52525b"
                {...props}
            />
            {error && <Text size="xs" color="error" style={styles.errorText}>{error}</Text>}
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        width: '100%',
        marginBottom: 16,
    },
    label: {
        marginBottom: 8,
        marginLeft: 4,
    },
    input: {
        backgroundColor: '#141414',
        borderRadius: 8,
        padding: 12,
        color: '#fafafa',
        fontSize: 16,
        borderWidth: 1,
        borderColor: '#1c1c1c',
    },
    inputError: {
        borderColor: '#ef4444',
    },
    errorText: {
        marginTop: 4,
        marginLeft: 4,
    },
});
