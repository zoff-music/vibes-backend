import React, { useState } from 'react';
import { View, StyleSheet, Modal, TouchableOpacity, Image } from 'react-native';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Text } from '../ui/Text';
import { useQueue } from '../../hooks/useQueue';
import { api } from '../../api/client';
import { parseISODuration, formatDuration } from '../../utils/time';

interface Props {
    roomId: string;
    isVisible: boolean;
    onClose: () => void;
}

export const AddToQueueModal: React.FC<Props> = ({ roomId, isVisible, onClose }) => {
    const [url, setUrl] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [previewVideo, setPreviewVideo] = useState<any>(null);
    const { addToQueue } = useQueue(roomId);

    const extractYoutubeId = (url: string) => {
        const regExp = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|\&v=)([^#\&\?]*).*/;
        const match = url.match(regExp);
        return (match && match[2].length === 11) ? match[2] : null;
    };

    const fetchVideoDetails = async (id: string) => {
        const [err, video] = await api.get('/youtube/videos/{id}', { id });
        if (err || !video) {
            setError('Failed to fetch video details');
            return null;
        }
        return video;
    };

    const handlePreview = async () => {
        setError(null);
        setPreviewVideo(null);
        const id = extractYoutubeId(url);
        if (!id) {
            setError('Invalid YouTube URL');
            return;
        }

        setIsLoading(true);
        const video = await fetchVideoDetails(id);
        setIsLoading(false);

        if (video) {
            setPreviewVideo(video);
        }
    };

    const handleAdd = async () => {
        if (!previewVideo) return;

        setIsLoading(true);
        const durationSec = parseISODuration(previewVideo.duration || 'PT0S');

        const success = await addToQueue(
            'youtube',
            previewVideo.id,
            previewVideo.title,
            previewVideo.thumbnailUrl,
            durationSec,
            previewVideo.channelTitle
        );

        setIsLoading(false);
        if (success) {
            setUrl('');
            setPreviewVideo(null);
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
                        title="Search"
                        onPress={handlePreview}
                        loading={isLoading && !previewVideo}
                        disabled={!url || (isLoading && !previewVideo)}
                        variant="secondary"
                        style={{ marginBottom: 16 }}
                    />

                    {previewVideo && (
                        <View style={styles.previewContainer}>
                            <Image source={{ uri: previewVideo.thumbnailUrl }} style={styles.thumbnail} />
                            <View style={styles.previewInfo}>
                                <Text bold numberOfLines={2}>{previewVideo.title}</Text>
                                <Text size="xs" color="muted">{previewVideo.channelTitle} • {formatDuration(parseISODuration(previewVideo.duration))}</Text>
                            </View>
                        </View>
                    )}

                    <Button
                        title="Add to Queue"
                        onPress={handleAdd}
                        loading={isLoading && !!previewVideo}
                        disabled={!previewVideo}
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
    },
    previewContainer: {
        flexDirection: 'row',
        backgroundColor: '#1c1c1c',
        borderRadius: 8,
        padding: 8,
        marginBottom: 24,
        alignItems: 'center',
    },
    thumbnail: {
        width: 80,
        height: 60,
        borderRadius: 4,
        marginRight: 12,
    },
    previewInfo: {
        flex: 1,
    }
});
