import React, { useState } from 'react';
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

    if (!isVisible) return null;

    return (
        <div
            className="fixed inset-0 bg-black/80 flex items-center justify-center p-6 z-50"
            onClick={onClose}
        >
            <div
                className="bg-background rounded-2xl p-6 border border-surfaceElevated max-w-md w-full"
                onClick={(e) => e.stopPropagation()}
            >
                <div className="flex items-center justify-between mb-6">
                    <Text size="lg" bold>Add Song</Text>
                    <button
                        onClick={onClose}
                        className="text-text-muted hover:text-text transition-colors"
                    >
                        Cancel
                    </button>
                </div>

                <Input
                    label="YouTube URL"
                    placeholder="https://www.youtube.com/watch?v=..."
                    value={url}
                    onChange={(e) => setUrl(e.target.value)}
                    error={error || undefined}
                    autoFocus
                />

                <div className="mb-6">
                    <Text size="xs" color="muted">
                        Tip: Paste a YouTube link to add it to the queue.
                    </Text>
                </div>

                <Button
                    title="Search"
                    onClick={handlePreview}
                    loading={isLoading && !previewVideo}
                    disabled={!url || (isLoading && !previewVideo)}
                    variant="secondary"
                    className="mb-4"
                />

                {previewVideo && (
                    <div className="flex bg-surfaceElevated rounded-lg p-2 mb-6 items-center">
                        <img
                            src={previewVideo.thumbnailUrl}
                            alt={previewVideo.title}
                            className="w-20 h-15 rounded object-cover mr-3"
                        />
                        <div className="flex-1 min-w-0">
                            <Text bold className="line-clamp-2">{previewVideo.title}</Text>
                            <Text size="xs" color="muted" className="mt-0.5">
                                {previewVideo.channelTitle} • {formatDuration(parseISODuration(previewVideo.duration))}
                            </Text>
                        </div>
                    </div>
                )}

                <Button
                    title="Add to Queue"
                    onClick={handleAdd}
                    loading={isLoading && !!previewVideo}
                    disabled={!previewVideo}
                />
            </div>
        </div>
    );
};
