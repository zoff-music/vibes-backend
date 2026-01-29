import { api, useQueue, useRoom } from '@vibez/api';
import {
  formatDuration,
  parseISODuration,
  type SourceType,
  usePlaybackStore,
  useQueueStore,
} from '@vibez/shared';
import {
  AlertCircleIcon,
  CheckIcon,
  CloseIcon,
  InfoIcon,
  PlusIcon,
  SearchIcon,
} from '@vibez/ui';
import React, { useEffect, useRef, useState } from 'react';
import { useAuthCache } from '../../hooks/useAuthCache';

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

interface SearchApiResult {
  id: string;
  title: string;
  channelTitle?: string;
  thumbnailUrl: string;
  duration?: string;
  url?: string;
  source?: string;
}

interface VideoApiResult {
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
  const { songs } = useQueueStore();
  const { currentSong } = usePlaybackStore();

  const hasSpotifySongs =
    songs.some((s) => s.sourceType === 'spotify') ||
    currentSong?.sourceType === 'spotify';

  const { providers, fetchProviders } = useAuthCache();
  // Ensure YouTube is first, then Spotify, then SoundCloud
  const orderedProviders: SourceType[] = ['youtube', 'spotify', 'soundcloud'];
  const providerList = orderedProviders.filter((p) =>
    (providers || []).includes(p),
  );

  const [selectedProvider, setSelectedProvider] =
    useState<SourceType>('youtube');

  useRoom(roomId);
  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    // Fetch available providers and authorizations via cache
    const loadData = async () => {
      await fetchProviders();
      // Logic to set default provider is handled by initial state + effect if needed,
      // but 'youtube' as default is good enough if available.
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

    let err: Error | null = null;
    let results: SearchApiResult[] | null = null;

    if (selectedProvider === 'youtube') {
      const [providerErr, providerResults] = await api.get('/youtube/search', {
        $search: { q: query },
      });
      err = providerErr;
      results = providerResults as SearchApiResult[] | null;
    } else if (selectedProvider === 'spotify') {
      const [providerErr, providerResults] = await api.get('/spotify/search', {
        $search: { q: query },
      });
      err = providerErr;
      results = providerResults as SearchApiResult[] | null;
    } else if (selectedProvider === 'soundcloud') {
      const [providerErr, providerResults] = await api.get(
        '/soundcloud/search',
        {
          $search: { q: query },
        },
      );
      err = providerErr;
      results = providerResults as SearchApiResult[] | null;
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
    const mappedResults: SearchResult[] = results.map((r) => ({
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
      api
        .get('/youtube/videos/{id}', { id: videoId })
        .then((result: [Error | null, VideoApiResult | null]) => {
          const [err, video] = result;
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
      selectedProvider,
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
      className="fixed inset-0 z-50 flex animate-fade-in items-start justify-center overflow-y-auto bg-black/70 pt-4 pb-safe backdrop-blur-md"
      role="dialog"
      aria-modal="true"
    >
      {/* Backdrop button */}
      <button
        type="button"
        className="fixed inset-0 h-full w-full cursor-pointer"
        onClick={onClose}
        aria-label="Close modal"
      />

      {/* Modal content */}
      <div className="panel-strong relative mx-4 w-full max-w-lg animate-scale-in rounded-[32px] p-7 shadow-[0_0_28px_rgba(255,46,151,0.25)]">
        {/* Header */}
        <div className="mb-6">
          <div className="mb-4 flex items-center justify-between">
            <div>
              <h2 className="font-display text-base text-theme">Add a Song</h2>
              <p className="mt-1 text-sm text-theme-muted">
                Search or paste a link
              </p>
            </div>
            <button
              onClick={onClose}
              className="cursor-pointer rounded-xl border border-transparent p-2 transition-colors hover:border-theme-strong hover:bg-theme-surface"
            >
              <CloseIcon className="h-5 w-5 text-theme-muted" />
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
                className={`cursor-pointer rounded-full px-4 py-1.5 font-medium text-xs transition-all ${
                  selectedProvider === p
                    ? 'border border-theme-subtle bg-theme-surface text-theme shadow-[0_0_12px_rgba(255,255,255,0.25)]'
                    : 'border border-transparent bg-theme-bg/10 text-theme-muted hover:bg-theme-surface/50 hover:text-theme'
                }`}
              >
                {p.charAt(0).toUpperCase() + p.slice(1)}
              </button>
            ))}
          </div>
        </div>

        {/* Spotify Disclaimer */}
        {selectedProvider === 'spotify' && !hasSpotifySongs && (
          <div className="mb-6 animate-slide-down rounded-2xl border border-secondary/30 bg-secondary/10 p-4 transition-all">
            <div className="flex gap-3">
              <div className="mt-0.5 text-secondary">
                <InfoIcon className="h-5 w-5" />
              </div>
              <p className="text-sm text-theme-muted leading-relaxed">
                <span className="font-display text-[10px] text-secondary">
                  Note:
                </span>{' '}
                By adding Spotify, viewers are required to have{' '}
                <span className="font-semibold text-theme">
                  Spotify Premium
                </span>{' '}
                to view content.
              </p>
            </div>
          </div>
        )}

        {/* Search Input */}
        <div className="relative mb-6">
          <div className="relative">
            {/* Auth Check Logic Removed: searching allowed without prior active source check */}

            <div className="absolute top-1/2 left-4 -translate-y-1/2 text-theme-muted">
              {isSearching ? (
                <div className="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
              ) : (
                <SearchIcon className="h-5 w-5" />
              )}
            </div>
            <input
              ref={inputRef}
              type="text"
              placeholder={`Search ${selectedProvider}...`}
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              onKeyDown={handleKeyDown}
              className="w-full rounded-2xl border border-theme bg-theme-surface py-4 pr-12 pl-12 text-base text-theme placeholder:text-theme-subtle focus:border-secondary focus:outline-hidden focus:ring-2 focus:ring-secondary/30"
              autoFocus
            />
            {searchQuery && (
              <button
                onClick={() => handleSearchChange('')}
                className="absolute top-1/2 right-3 -translate-y-1/2 cursor-pointer rounded-lg p-1.5 transition-colors hover:bg-theme-surface"
              >
                <CloseIcon
                  className="h-5 w-5 text-theme-subtle"
                  strokeWidth={2}
                />
              </button>
            )}
          </div>

          {error && (
            <div className="mt-3 flex animate-slide-down items-start gap-2 text-error text-sm">
              <AlertCircleIcon className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Search Results Dropdown */}
          {showResults &&
            searchResults.length > 0 &&
            !isLoading &&
            !justAdded && (
              <div className="mt-2 max-h-96 w-full animate-scale-in overflow-hidden overflow-y-auto rounded-2xl border border-theme bg-theme-surface shadow-[0_0_24px_rgba(255,46,151,0.25)]">
                {searchResults.map((result, index) => (
                  <button
                    key={result.id}
                    onClick={() => handleSelectResult(result)}
                    className={`flex w-full cursor-pointer gap-3 p-4 text-left transition-all hover:bg-theme ${
                      index > 0 ? 'border-theme border-t' : ''
                    }`}
                  >
                    <div className="relative shrink-0">
                      <img
                        src={result.thumbnailUrl}
                        alt={result.title}
                        className="h-20 w-28 rounded-xl border border-theme bg-theme-surface object-cover"
                      />
                      {result.duration && (
                        <div className="absolute right-1.5 bottom-1.5 rounded-md bg-theme px-2 py-0.5 text-[10px] text-theme backdrop-blur-sm">
                          {formatDuration(parseISODuration(result.duration))}
                        </div>
                      )}
                    </div>
                    <div className="flex min-w-0 flex-1 flex-col justify-center">
                      <h4 className="mb-1.5 line-clamp-2 text-sm text-theme leading-snug">
                        {result.title}
                      </h4>
                      <p className="line-clamp-1 text-theme-muted text-xs">
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
          <div className="animate-scale-in rounded-2xl border border-theme bg-theme-surface p-8 text-center">
            <div className="mb-3 inline-flex h-14 w-14 items-center justify-center rounded-2xl border border-theme bg-theme">
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-white/30 border-t-white" />
            </div>
            <p className="text-sm text-theme-muted">Loading preview...</p>
          </div>
        )}

        {/* Video Preview */}
        {previewVideo && !justAdded && (
          <div className="mb-6 animate-scale-in rounded-2xl border border-theme bg-theme-surface p-4">
            <div className="flex gap-4">
              <div className="relative shrink-0">
                <img
                  src={previewVideo.thumbnailUrl}
                  alt={previewVideo.title}
                  className="h-24 w-32 rounded-xl border border-theme bg-theme-surface object-cover"
                />
                <div className="absolute right-1.5 bottom-1.5 rounded-md bg-theme px-2 py-0.5 text-[10px] text-theme backdrop-blur-sm">
                  {formatDuration(parseISODuration(previewVideo.duration))}
                </div>
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="mb-2 line-clamp-2 text-sm text-theme">
                  {previewVideo.title}
                </h3>
                <p className="line-clamp-1 text-theme-muted text-xs">
                  {previewVideo.artist}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Success State */}
        {justAdded && (
          <div className="animate-scale-in rounded-2xl border border-secondary/40 bg-secondary/10 p-10 text-center">
            <div className="mb-4 inline-flex h-20 w-20 items-center justify-center rounded-2xl border border-secondary/40 bg-secondary/20">
              <CheckIcon className="h-10 w-10 text-secondary" />
            </div>
            <h3 className="mb-2 font-display text-base text-theme">
              Added to Queue!
            </h3>
            <p className="mb-1 text-sm text-theme-muted">
              Everyone will hear it soon
            </p>
            <p className="jp-art text-theme-subtle text-xs">追加されました</p>
          </div>
        )}

        {/* Action Buttons */}
        {previewVideo && !justAdded && (
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="flex-1 cursor-pointer rounded-xl border border-theme bg-theme-surface py-3 text-theme-muted text-xs transition-all hover:border-theme-strong active:scale-[0.98]"
            >
              Cancel
            </button>
            <button
              onClick={handleAdd}
              disabled={isLoading}
              className="flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-xl bg-primary py-3 text-white text-xs shadow-[0_0_18px_rgba(255,46,151,0.4)] transition-all hover:bg-primary-muted active:scale-[0.98] disabled:bg-theme-surface disabled:text-theme-subtle"
            >
              {isLoading ? (
                <>
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                  <span>Adding...</span>
                </>
              ) : (
                <>
                  <PlusIcon className="h-5 w-5" />
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
