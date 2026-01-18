import React, { useState, useEffect } from 'react';
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
    const [justAdded, setJustAdded] = useState(false);
    const { addToQueue } = useQueue(roomId);

    useEffect(() => {
        if (!isVisible) {
            setTimeout(() => {
                setUrl('');
                setPreviewVideo(null);
                setError(null);
                setJustAdded(false);
            }, 300);
        }
    }, [isVisible]);

    const extractYoutubeId = (url: string) => {
        const regExp = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|\&v=)([^#\&\?]*).*/;
        const match = url.match(regExp);
        return (match && match[2].length === 11) ? match[2] : null;
    };

    const fetchVideoDetails = async (id: string) => {
        const [err, video] = await api.get('/youtube/videos/{id}', { id });
        if (err || !video) {
            setError('Could not find that video');
            return null;
        }
        return video;
    };

    const handleUrlChange = async (newUrl: string) => {
        setUrl(newUrl);
        setError(null);

        const id = extractYoutubeId(newUrl);
        if (id) {
            setIsLoading(true);
            setPreviewVideo(null);
            const video = await fetchVideoDetails(id);
            setIsLoading(false);
            if (video) {
                setPreviewVideo(video);
            }
        } else {
            setPreviewVideo(null);
        }
    };

    const handleAdd = async () => {
        if (!previewVideo || justAdded) return;

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
            setJustAdded(true);
            setTimeout(() => {
                onClose();
            }, 800);
        } else {
            setError('Failed to add song to queue');
        }
    };

    if (!isVisible) return null;

    return (
        <div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50 animate-fade-in"
            onClick={onClose}
        >
            <div
                className="glass-elevated rounded-2xl p-6 max-w-lg w-full animate-scale-in"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between mb-6">
                    <div>
                        <h2 className="text-xl font-bold">Add a Song</h2>
                        <p className="text-sm text-text-muted mt-1">Paste a YouTube link</p>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-2 hover:bg-surfaceHover rounded-lg transition-colors"
                    >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                {/* URL Input */}
                <div className="mb-6">
                    <div className="relative">
                        <input
                            type="text"
                            placeholder="https://youtube.com/watch?v=..."
                            value={url}
                            onChange={(e) => handleUrlChange(e.target.value)}
                            className="w-full bg-surfaceElevated/50 backdrop-blur-sm rounded-xl px-4 py-3.5 pr-12 text-base text-white placeholder:text-text-subtle border border-border focus:outline-none focus:border-primary/50 focus:ring-2 focus:ring-primary/20 transition-all"
                            autoFocus
                        />
                        {url && (
                            <button
                                onClick={() => handleUrlChange('')}
                                className="absolute right-3 top-1/2 -translate-y-1/2 p-1.5 hover:bg-surfaceHover rounded-lg transition-colors"
                            >
                                <svg className="w-4 h-4 text-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        )}
                    </div>

                    {error && (
                        <div className="mt-3 flex items-start gap-2 text-error text-sm animate-slide-down">
                            <svg className="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            <span>{error}</span>
                        </div>
                    )}
                </div>

                {/* Loading State */}
                {isLoading && !previewVideo && (
                    <div className="glass rounded-xl p-8 text-center animate-scale-in">
                        <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-surfaceElevated mb-3">
                            <div className="w-6 h-6 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
                        </div>
                        <p className="text-sm text-text-muted">Loading preview...</p>
                    </div>
                )}

                {/* Video Preview */}
                {previewVideo && !justAdded && (
                    <div className="glass rounded-xl p-4 mb-6 animate-scale-in">
                        <div className="flex gap-4">
                            <div className="relative flex-shrink-0">
                                <img
                                    src={previewVideo.thumbnailUrl}
                                    alt={previewVideo.title}
                                    className="w-32 h-20 rounded-lg object-cover bg-surfaceElevated ring-1 ring-border"
                                />
                                <div className="absolute bottom-1.5 right-1.5 bg-black/80 backdrop-blur-sm px-1.5 py-0.5 rounded text-xs font-medium">
                                    {formatDuration(parseISODuration(previewVideo.duration))}
                                </div>
                            </div>
                            <div className="flex-1 min-w-0">
                                <h3 className="font-semibold line-clamp-2 mb-2">
                                    {previewVideo.title}
                                </h3>
                                <p className="text-sm text-text-muted line-clamp-1">
                                    {previewVideo.channelTitle}
                                </p>
                            </div>
                        </div>
                    </div>
                )}

                {/* Success State */}
                {justAdded && (
                    <div className="glass rounded-xl p-8 text-center animate-scale-in">
                        <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-success/10 mb-4">
                            <svg className="w-8 h-8 text-success" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                            </svg>
                        </div>
                        <h3 className="text-lg font-semibold mb-1">Added to Queue!</h3>
                        <p className="text-sm text-text-muted">Everyone will hear it soon</p>
                    </div>
                )}

                {/* Action Buttons */}
                {previewVideo && !justAdded && (
                    <div className="flex gap-3">
                        <button
                            onClick={onClose}
                            className="flex-1 glass py-3 rounded-xl font-medium hover:bg-surfaceHover transition-all active:scale-[0.98]"
                        >
                            Cancel
                        </button>
                        <button
                            onClick={handleAdd}
                            disabled={isLoading}
                            className="flex-1 bg-primary hover:bg-primary-muted disabled:bg-surface disabled:text-text-subtle text-white py-3 rounded-xl font-medium transition-all hover:shadow-glow active:scale-[0.98] flex items-center justify-center gap-2"
                        >
                            {isLoading ? (
                                <>
                                    <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                                    <span>Adding...</span>
                                </>
                            ) : (
                                <>
                                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                                    </svg>
                                    <span>Add to Queue</span>
                                </>
                            )}
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
};
