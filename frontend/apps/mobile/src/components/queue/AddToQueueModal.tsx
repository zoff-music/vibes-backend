import React, { useState } from 'react';
import { View, StyleSheet, Modal, TouchableOpacity } from 'react-native';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Text } from '../ui/Text';
import { useQueue } from '../../hooks/useQueue';

interface Props {
    roomId: string;
    isVisible: boolean;
    onClose: () => void;
}

export const AddToQueueModal: React.FC<Props> = ({ roomId, isVisible, onClose }) => {
    const [url, setUrl] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const { addToQueue } = useQueue(roomId);

    const extractYoutubeId = (url: string) => {
        const regExp = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|\&v=)([^#\&\?]*).*/;
        const match = url.match(regExp);
        return (match && match[2].length === 11) ? match[2] : null;
    };

    const handleAdd = async () => {
        setError(null);
        const id = extractYoutubeId(url);
        if (!id) {
            setError('Invalid YouTube URL');
            return;
        }

        setIsLoading(true);
        // For MVP, we'll use some placeholder metadata since we don't have a resolver yet
        // In a real app, this would be fetched from YouTube API (client or server side)
        const success = await addToQueue(
            'youtube',
            id,
            'YouTube Video', // Placeholder title
            `https://img.youtube.com/vi/${id}/mqdefault.jpg`,
            300, // Placeholder duration (5 mins)
            'Unknown Artist'
        );

        setIsLoading(false);
        if (success) {
            setUrl('');
            onClose();
        } else {
            setError('Failed to add song to queue');
        }
    };

    return (
        <Modal
            visible={isVisible}
            transparent
            animationType="fade"
            onRequestClose={onClose}
        >
            <TouchableOpacity
                style={styles.overlay}
                activeOpacity={1}
                onPress={onClose}
            >
                <TouchableOpacity
                    style={styles.container}
                    activeOpacity={1}
                    onPress={(e) => e.stopPropagation()}
                >
                    <View style={styles.header}>
                        <Text size="lg" bold>Add Song</Text>
                        <TouchableOpacity onPress={onClose}>
                            <Text color="muted">Cancel</Text>
                        </TouchableOpacity>
                    </View>

                    <Input
                        label="YouTube URL"
                        placeholder="https://www.youtube.com/watch?v=..."
                        value={url}
                        onChangeText={setUrl}
                        error={error || undefined}
                        autoFocus
                    />

                    <View style={styles.tips}>
                        <Text size="xs" color="muted">Tip: Paste a YouTube link to add it to the queue.</Text>
                    </View>

                    <Button
                        title="Add to Queue"
                        onPress={handleAdd}
                        loading={isLoading}
                        disabled={!url}
                    />
                </TouchableOpacity>
            </TouchableOpacity>
        </Modal>
    );
};

const styles = StyleSheet.create({
    overlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.8)',
        justifyContent: 'center',
        padding: 24,
    },
    container: {
        backgroundColor: '#0a0a0a',
        borderRadius: 16,
        padding: 24,
        borderWidth: 1,
        borderColor: '#1c1c1c',
    },
    header: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 24,
    },
    tips: {
        marginBottom: 24,
    }
});
