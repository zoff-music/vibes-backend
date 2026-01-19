import { api } from '@vibez/api';
import { formatDuration, parseISODuration } from '@vibez/shared';
import React, { useEffect, useRef, useState } from 'react';
import { useAuthCache } from '../../hooks/useAuthCache';
import { useQueue } from '../../hooks/useQueue';
import { useRoom } from '../../hooks/useRoom';

interface Props {
  roomId: string;
  isVisible: boolean;
  onClose: () => void;
}

interface SearchResult {
  id: string;
  title: string;
  artist: string;
  thumbnailUrl: string;
  duration?: string;
  url?: string;
  source?: string;
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

  const { providers, fetchProviders } = useAuthCache();
  const providerList = providers || [];

  const [selectedProvider, setSelectedProvider] = useState<string>('youtube');

  useRoom(roomId);
  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    // Fetch available providers and authorizations via cache
    const loadData = async () => {
      const pData = await fetchProviders();
      // Set default selected provider if not set or invalid
      if (
        pData.length > 0 &&
        !pData.includes(selectedProvider) &&
        selectedProvider === 'youtube' &&
        !pData.includes('youtube')
      ) {
        if (pData.includes('spotify')) setSelectedProvider('spotify');
        else setSelectedProvider(pData[0]);
      } else if (
        pData.length > 0 &&
        selectedProvider === 'youtube' &&
        pData.includes('youtube')
      ) {
        // keep youtube
      }
    };
    loadData();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

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

    let err, results;

    if (selectedProvider === 'youtube') {
      [err, results] = await api.get('/youtube/search', {
        $search: { q: query },
      });
    } else if (selectedProvider === 'spotify') {
      [err, results] = await api.get('/spotify/search', {
        $search: { q: query },
      });
    } else if (selectedProvider === 'soundcloud') {
      [err, results] = await api.get('/soundcloud/search', {
        $search: { q: query },
      });
    }

    setIsSearching(false);

    if (err || !results) {
      setError('Search failed. Please try again.');
      setSearchResults([]);
      setShowResults(false);
      return;
    }

    // Backend returns MusicTracks with { id, title, artist, duration, thumbnail, url }
    // We map it to SearchResult { id, title, artist, thumbnailUrl, duration, url }
    const mappedResults: SearchResult[] = results.map((r: any) => ({
      id: r.id,
      title: r.title,
      artist: r.channelTitle || 'Unknown',
      thumbnailUrl: r.thumbnailUrl,
      duration: r.duration,
      url: r.url, // Backend might not send this, but keeping it optional
      source: r.source,
    }));

    setSearchResults(mappedResults);
    setShowResults(true);
  };

  const handleSearchChange = (query: string) => {
    setSearchQuery(query);
    setError(null);
    setPreviewVideo(null);

    // Check if it's a YouTube URL (only if strictly YouTube provider or maybe auto-detect?)
    // For now let's keep it simple: if generic URL detected, maybe switch to generic "add by URL"?
    // But existing logic was specific to YouTube ID extraction.
    const videoId = extractYoutubeId(query);
    if (videoId && selectedProvider === 'youtube') {
      setShowResults(false);
      setIsSearching(true);
      api.get('/youtube/videos/{id}', { id: videoId }).then(([err, video]) => {
        setIsSearching(false);
        if (err || !video) {
          setError('Could not find that video');
          return;
        }
        // Map YouTube video to generic result
        setPreviewVideo({
          id: video.id,
          title: video.title,
          artist: video.channelTitle,
          thumbnailUrl: video.thumbnailUrl,
          duration: video.duration,
        });
      });
      return;
    }

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

    // Determine sourceType from selectedProvider (which is 'youtube', 'spotify', etc.)
    // Assuming selectedProvider matches sourceType strings.
    const success = await addToQueue(
      selectedProvider as any, // Cast to any or SourceType
      result.id,
      result.title,
      result.thumbnailUrl,
      durationSec,
      result.artist,
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

  const handleAdd = async () => {
    if (!previewVideo || justAdded) return;
    handleSelectResult(previewVideo);
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
        <div className="mb-6">
          <div className="mb-4 flex items-center justify-between">
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

          {/* Provider Tabs */}
          <div className="scrollbar-none flex gap-2 overflow-x-auto pb-2">
            {providerList.map((p) => (
              <button
                key={p}
                onClick={() => {
                  setSelectedProvider(p);
                  setSearchResults([]);
                  setSearchQuery('');
                  setPreviewVideo(null);
                }}
                className={`rounded-full px-4 py-1.5 font-bold text-sm transition-all ${
                  selectedProvider === p
                    ? 'bg-ink text-white shadow-lg'
                    : 'bg-surface text-ink/60 hover:bg-ink/5'
                }`}
              >
                {p.charAt(0).toUpperCase() + p.slice(1)}
              </button>
            ))}
          </div>
        </div>

        {/* Search Input */}
        <div className="relative mb-6">
          <div className="relative">
            {/* Auth Check Logic Removed: searching allowed without prior active source check */}

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
              placeholder={`Search ${selectedProvider}...`}
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
                      {result.artist}
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
                  {previewVideo.artist}
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
