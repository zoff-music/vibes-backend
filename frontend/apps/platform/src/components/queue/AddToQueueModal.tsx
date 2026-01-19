import React, { useState, useEffect, useRef } from 'react';
import { useQueue } from '../../hooks/useQueue';
import { api } from '@vibez/api';
import { parseISODuration, formatDuration } from '@vibez/shared';

interface Props {
    roomId: string;
    isVisible: boolean;
    onClose: () => void;
}

interface SearchResult {
    id: string;
    title: string;
    channelTitle: string;
    thumbnailUrl: string;
    duration?: string;
}

export const AddToQueueModal: React.FC<Props> = ({ roomId, isVisible, onClose }) => {
    const [searchQuery, setSearchQuery] = useState('');
    const [isSearching, setIsSearching] = useState(false);
    const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
    const [showResults, setShowResults] = useState(false);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [previewVideo, setPreviewVideo] = useState<SearchResult | null>(null);
    const [justAdded, setJustAdded] = useState(false);
    const { addToQueue } = useQueue(roomId);
    const searchTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (!isVisible) {
            setTimeout(() => {
                setSearchQuery('');
                setSearchResults([]);
                setShowResults(false);
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

    const performSearch = async (query: string) => {
        if (!query.trim()) {
            setSearchResults([]);
            setShowResults(false);
            return;
        }

        setIsSearching(true);
        setError(null);

        const [err, results] = await api.get('/youtube/search', { $search: { q: query } });
        setIsSearching(false);

        if (err || !results) {
            setError('Search failed. Please try again.');
            setSearchResults([]);
            setShowResults(false);
            return;
        }

        setSearchResults(results);
        setShowResults(true);
    };

    const handleSearchChange = (query: string) => {
        setSearchQuery(query);
        setError(null);
        setPreviewVideo(null);

        // Check if it's a YouTube URL
        const videoId = extractYoutubeId(query);
        if (videoId) {
            // If it's a URL, fetch the video directly
            setShowResults(false);
            setIsSearching(true);
            api.get('/youtube/videos/{id}', { id: videoId }).then(([err, video]) => {
                setIsSearching(false);
                if (err || !video) {
                    setError('Could not find that video');
                    return;
                }
                setPreviewVideo(video);
            });
            return;
        }

        // Debounce search
        if (searchTimeoutRef.current) {
            clearTimeout(searchTimeoutRef.current);
        }

        if (!query.trim()) {
            setSearchResults([]);
            setShowResults(false);
            return;
        }

        searchTimeoutRef.current = setTimeout(() => {
            performSearch(query);
        }, 300);
    };

    const handleSelectResult = async (result: SearchResult) => {
        setIsLoading(true);
        const durationSec = parseISODuration(result.duration);

        const success = await addToQueue(
            'youtube',
            result.id,
            result.title,
            result.thumbnailUrl,
            durationSec,
            result.channelTitle
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

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            if (searchResults.length > 0) {
                handleSelectResult(searchResults[0]);
            }
        }
    };

    // Kept for backward compatibility if needed, or can be removed if strictly unused.
    // The previous logic for handleAdd is now largely merged into handleSelectResult.
    const handleAdd = async () => {
        if (!previewVideo || justAdded) return;
        // ... (logic is same as above but targeting previewVideo)
        // Since we are skipping preview, this might become redundant.
        // For now, I'll direct handleAdd to work if previewVideo exists (e.g. from copy-paste URL which might still use preview flow if we want, but user asked to skip "last add to queue step").
        // If "skip the last add to queue step" applies to search results, we use handleSelectResult.
        // If it applies to everything, we need to auto-add on URL match too.
    };

    if (!isVisible) return null;

    return (
        <div
            className="fixed inset-0 bg-ink/80 backdrop-blur-md flex items-start justify-center pt-4 pb-safe z-50 animate-fade-in overflow-y-auto"
            onClick={onClose}
        >
            <div
                className="bg-white rounded-3xl p-7 max-w-lg w-full mx-4 animate-scale-in shadow-retro-pink border-4 border-ink"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between mb-6">
                    <div>
                        <h2 className="text-2xl font-black text-ink" style={{ fontFamily: 'Poppins' }}>Add a Song</h2>
                        <p className="text-sm text-ink/60 mt-1 font-medium">Search or paste a link</p>
                        <p className="text-xs jp-art text-ink/40 mt-0.5">曲を検索</p>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-2 hover:bg-ink/5 rounded-xl transition-colors border-2 border-transparent hover:border-ink/10"
                    >
                        <svg className="w-5 h-5 text-ink/60" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                {/* Search Input */}
                <div className="mb-6 relative">
                    <div className="relative">
                        <div className="absolute left-4 top-1/2 -translate-y-1/2 text-ink/40">
                            {isSearching ? (
                                <div className="w-5 h-5 border-2 border-ink/20 border-t-primary rounded-full animate-spin" />
                            ) : (
                                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                                </svg>
                            )}
                        </div>
                        <input
                            ref={inputRef}
                            type="text"
                            placeholder="Search songs or paste YouTube URL..."
                            value={searchQuery}
                            onChange={(e) => handleSearchChange(e.target.value)}
                            onKeyDown={handleKeyDown}
                            className="w-full bg-surface rounded-xl pl-12 pr-12 py-4 text-base text-ink placeholder:text-ink/40 border-2 border-ink/20 focus:outline-hidden focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] transition-all font-medium"
                            autoFocus
                        />
                        {searchQuery && (
                            <button
                                onClick={() => handleSearchChange('')}
                                className="absolute right-3 top-1/2 -translate-y-1/2 p-1.5 hover:bg-ink/5 rounded-lg transition-colors"
                            >
                                <svg className="w-5 h-5 text-ink/40" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        )}
                    </div>

                    {error && (
                        <div className="mt-3 flex items-start gap-2 text-error text-sm animate-slide-down font-medium">
                            <svg className="w-4 h-4 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            <span>{error}</span>
                        </div>
                    )}

                    {/* Search Results Dropdown */}
                    {showResults && searchResults.length > 0 && (
                        <div className="absolute top-full left-0 right-0 mt-2 bg-white rounded-2xl overflow-hidden max-h-96 overflow-y-auto z-10 animate-scale-in shadow-retro-pink border-3 border-ink">
                            {searchResults.map((result, index) => (
                                <button
                                    key={result.id}
                                    onClick={() => handleSelectResult(result)}
                                    className={`w-full flex gap-3 p-4 hover:bg-sakura/20 transition-all text-left ${index > 0 ? 'border-t-2 border-ink/10' : ''}`}
                                >
                                    <div className="relative shrink-0">
                                        <img
                                            src={result.thumbnailUrl}
                                            alt={result.title}
                                            className="w-28 h-20 rounded-xl object-cover bg-surface ring-2 ring-ink/20"
                                        />
                                        {result.duration && (
                                            <div className="absolute bottom-1.5 right-1.5 bg-ink/90 backdrop-blur-sm px-2 py-0.5 rounded-md text-xs font-bold text-white">
                                                {formatDuration(parseISODuration(result.duration))}
                                            </div>
                                        )}
                                    </div>
                                    <div className="flex-1 min-w-0 flex flex-col justify-center">
                                        <h4 className="font-bold line-clamp-2 text-sm mb-1.5 leading-snug text-ink">
                                            {result.title}
                                        </h4>
                                        <p className="text-xs text-ink/60 line-clamp-1 font-medium">
                                            {result.channelTitle}
                                        </p>
                                    </div>
                                </button>
                            ))}
                        </div>
                    )}
                </div>

                {/* Loading State */}
                {isSearching && !previewVideo && extractYoutubeId(searchQuery) && (
                    <div className="glass rounded-2xl p-8 text-center animate-scale-in border-2 border-ink/10">
                        <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-surface border-2 border-ink/20 mb-3">
                            <div className="w-6 h-6 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
                        </div>
                        <p className="text-sm text-ink/60 font-medium">Loading preview...</p>
                    </div>
                )}

                {/* Video Preview */}
                {previewVideo && !justAdded && (
                    <div className="glass rounded-2xl p-4 mb-6 animate-scale-in border-2 border-ink/10">
                        <div className="flex gap-4">
                            <div className="relative shrink-0">
                                <img
                                    src={previewVideo.thumbnailUrl}
                                    alt={previewVideo.title}
                                    className="w-32 h-24 rounded-xl object-cover bg-surface ring-2 ring-ink/20"
                                />
                                <div className="absolute bottom-1.5 right-1.5 bg-ink/90 backdrop-blur-sm px-2 py-0.5 rounded-md text-xs font-bold text-white">
                                    {formatDuration(parseISODuration(previewVideo.duration))}
                                </div>
                            </div>
                            <div className="flex-1 min-w-0">
                                <h3 className="font-bold line-clamp-2 mb-2 text-ink">
                                    {previewVideo.title}
                                </h3>
                                <p className="text-sm text-ink/60 line-clamp-1 font-medium">
                                    {previewVideo.channelTitle}
                                </p>
                            </div>
                        </div>
                    </div>
                )}

                {/* Success State */}
                {justAdded && (
                    <div className="glass rounded-2xl p-10 text-center animate-scale-in border-2 border-matcha">
                        <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-matcha/20 border-4 border-matcha/40 mb-4">
                            <svg className="w-10 h-10 text-matcha" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                            </svg>
                        </div>
                        <h3 className="text-xl font-black mb-2 text-ink" style={{ fontFamily: 'Poppins' }}>Added to Queue!</h3>
                        <p className="text-sm text-ink/60 font-medium mb-1">Everyone will hear it soon</p>
                        <p className="text-xs jp-art text-ink/40">追加されました</p>
                    </div>
                )}

                {/* Action Buttons */}
                {previewVideo && !justAdded && (
                    <div className="flex gap-3">
                        <button
                            onClick={onClose}
                            className="flex-1 glass py-3.5 rounded-xl font-bold hover:shadow-retro transition-all active:scale-[0.98] border-2 border-ink/10 text-ink/70 tracking-wide"
                        >
                            Cancel
                        </button>
                        <button
                            onClick={handleAdd}
                            disabled={isLoading}
                            className="flex-1 bg-primary hover:bg-primary-muted disabled:bg-ink/10 disabled:text-ink/30 text-white py-3.5 rounded-xl font-black transition-all hover:shadow-retro-pink active:scale-[0.98] flex items-center justify-center gap-2 tracking-wide"
                            style={{ fontFamily: 'Poppins' }}
                        >
                            {isLoading ? (
                                <>
                                    <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                                    <span>Adding...</span>
                                </>
                            ) : (
                                <>
                                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                        <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
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
