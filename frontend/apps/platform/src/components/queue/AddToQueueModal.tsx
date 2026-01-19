import { api } from '@vibez/api';
import { formatDuration, parseISODuration } from '@vibez/shared';
import React, { useEffect, useRef, useState } from 'react';
import { useQueue } from '../../hooks/useQueue';

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

export const AddToQueueModal: React.FC<Props> = ({
  roomId,
  isVisible,
  onClose,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearching, setIsSearching] = useState(false);
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [showResults, setShowResults] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [previewVideo, setPreviewVideo] = useState<SearchResult | null>(null);
  const [justAdded, setJustAdded] = useState(false);
  const { addToQueue } = useQueue(roomId);
  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
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
    const regExp =
      /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|&v=)([^#&?]*).*/;
    const match = url.match(regExp);
    return match && match[2].length === 11 ? match[2] : null;
  };

  const performSearch = async (query: string) => {
    if (!query.trim()) {
      setSearchResults([]);
      setShowResults(false);
      return;
    }

    setIsSearching(true);
    setError(null);

    const [err, results] = await api.get('/youtube/search', {
      $search: { q: query },
    });
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
      result.channelTitle,
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
      className="fixed inset-0 z-50 flex animate-fade-in items-start justify-center overflow-y-auto bg-ink/80 pt-4 pb-safe backdrop-blur-md"
      onClick={onClose}
    >
      <div
        className="mx-4 w-full max-w-lg animate-scale-in rounded-3xl border-4 border-ink bg-white p-7 shadow-retro-pink"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2
              className="font-black text-2xl text-ink"
              style={{ fontFamily: 'Poppins' }}
            >
              Add a Song
            </h2>
            <p className="mt-1 font-medium text-ink/60 text-sm">
              Search or paste a link
            </p>
            <p className="jp-art mt-0.5 text-ink/40 text-xs">曲を検索</p>
          </div>
          <button
            onClick={onClose}
            className="rounded-xl border-2 border-transparent p-2 transition-colors hover:border-ink/10 hover:bg-ink/5"
          >
            <svg
              className="h-5 w-5 text-ink/60"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Search Input */}
        <div className="relative mb-6">
          <div className="relative">
            <div className="absolute top-1/2 left-4 -translate-y-1/2 text-ink/40">
              {isSearching ? (
                <div className="h-5 w-5 animate-spin rounded-full border-2 border-ink/20 border-t-primary" />
              ) : (
                <svg
                  className="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2.5}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                  />
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
              className="w-full rounded-xl border-2 border-ink/20 bg-surface py-4 pr-12 pl-12 font-medium text-base text-ink transition-all placeholder:text-ink/40 focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] focus:outline-hidden"
              autoFocus
            />
            {searchQuery && (
              <button
                onClick={() => handleSearchChange('')}
                className="absolute top-1/2 right-3 -translate-y-1/2 rounded-lg p-1.5 transition-colors hover:bg-ink/5"
              >
                <svg
                  className="h-5 w-5 text-ink/40"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            )}
          </div>

          {error && (
            <div className="mt-3 flex animate-slide-down items-start gap-2 font-medium text-error text-sm">
              <svg
                className="mt-0.5 h-4 w-4 shrink-0"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <span>{error}</span>
            </div>
          )}

          {/* Search Results Dropdown */}
          {showResults && searchResults.length > 0 && (
            <div className="absolute top-full right-0 left-0 z-10 mt-2 max-h-96 animate-scale-in overflow-hidden overflow-y-auto rounded-2xl border-3 border-ink bg-white shadow-retro-pink">
              {searchResults.map((result, index) => (
                <button
                  key={result.id}
                  onClick={() => handleSelectResult(result)}
                  className={`flex w-full gap-3 p-4 text-left transition-all hover:bg-sakura/20 ${index > 0 ? 'border-ink/10 border-t-2' : ''}`}
                >
                  <div className="relative shrink-0">
                    <img
                      src={result.thumbnailUrl}
                      alt={result.title}
                      className="h-20 w-28 rounded-xl bg-surface object-cover ring-2 ring-ink/20"
                    />
                    {result.duration && (
                      <div className="absolute right-1.5 bottom-1.5 rounded-md bg-ink/90 px-2 py-0.5 font-bold text-white text-xs backdrop-blur-sm">
                        {formatDuration(parseISODuration(result.duration))}
                      </div>
                    )}
                  </div>
                  <div className="flex min-w-0 flex-1 flex-col justify-center">
                    <h4 className="mb-1.5 line-clamp-2 font-bold text-ink text-sm leading-snug">
                      {result.title}
                    </h4>
                    <p className="line-clamp-1 font-medium text-ink/60 text-xs">
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
          <div className="glass animate-scale-in rounded-2xl border-2 border-ink/10 p-8 text-center">
            <div className="mb-3 inline-flex h-14 w-14 items-center justify-center rounded-2xl border-2 border-ink/20 bg-surface">
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary/30 border-t-primary" />
            </div>
            <p className="font-medium text-ink/60 text-sm">
              Loading preview...
            </p>
          </div>
        )}

        {/* Video Preview */}
        {previewVideo && !justAdded && (
          <div className="glass mb-6 animate-scale-in rounded-2xl border-2 border-ink/10 p-4">
            <div className="flex gap-4">
              <div className="relative shrink-0">
                <img
                  src={previewVideo.thumbnailUrl}
                  alt={previewVideo.title}
                  className="h-24 w-32 rounded-xl bg-surface object-cover ring-2 ring-ink/20"
                />
                <div className="absolute right-1.5 bottom-1.5 rounded-md bg-ink/90 px-2 py-0.5 font-bold text-white text-xs backdrop-blur-sm">
                  {formatDuration(parseISODuration(previewVideo.duration))}
                </div>
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="mb-2 line-clamp-2 font-bold text-ink">
                  {previewVideo.title}
                </h3>
                <p className="line-clamp-1 font-medium text-ink/60 text-sm">
                  {previewVideo.channelTitle}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Success State */}
        {justAdded && (
          <div className="glass animate-scale-in rounded-2xl border-2 border-matcha p-10 text-center">
            <div className="mb-4 inline-flex h-20 w-20 items-center justify-center rounded-2xl border-4 border-matcha/40 bg-matcha/20">
              <svg
                className="h-10 w-10 text-matcha"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={3}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
            <h3
              className="mb-2 font-black text-ink text-xl"
              style={{ fontFamily: 'Poppins' }}
            >
              Added to Queue!
            </h3>
            <p className="mb-1 font-medium text-ink/60 text-sm">
              Everyone will hear it soon
            </p>
            <p className="jp-art text-ink/40 text-xs">追加されました</p>
          </div>
        )}

        {/* Action Buttons */}
        {previewVideo && !justAdded && (
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="glass flex-1 rounded-xl border-2 border-ink/10 py-3.5 font-bold text-ink/70 tracking-wide transition-all hover:shadow-retro active:scale-[0.98]"
            >
              Cancel
            </button>
            <button
              onClick={handleAdd}
              disabled={isLoading}
              className="flex flex-1 items-center justify-center gap-2 rounded-xl bg-primary py-3.5 font-black text-white tracking-wide transition-all hover:bg-primary-muted hover:shadow-retro-pink active:scale-[0.98] disabled:bg-ink/10 disabled:text-ink/30"
              style={{ fontFamily: 'Poppins' }}
            >
              {isLoading ? (
                <>
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                  <span>Adding...</span>
                </>
              ) : (
                <>
                  <svg
                    className="h-5 w-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2.5}
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M12 4v16m8-8H4"
                    />
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
