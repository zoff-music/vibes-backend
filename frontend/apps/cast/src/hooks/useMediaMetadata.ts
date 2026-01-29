import type { Song } from '@vibez/shared';
import { safeWrap } from '@vibez/shared';
import type { framework } from 'chromecast-caf-receiver';
import { useCallback } from 'react';

interface ExtendedPlayerManager extends framework.PlayerManager {
  broadcastStatus(includeMediaStatus: boolean): void;
}

export function useMediaMetadata() {
  const updateMediaMetadata = useCallback((song: Song) => {
    if (!window.cast || !window.cast.framework) return;

    const context = window.cast.framework.CastReceiverContext.getInstance();
    const playerManager = context.getPlayerManager();
    if (!playerManager) return;

    const [err] = safeWrap(() => {
      const mediaInfo =
        playerManager.getMediaInformation() ||
        new cast.framework.messages.MediaInformation();

      const metadata = new cast.framework.messages.MusicTrackMediaMetadata();
      metadata.title = song.title;
      metadata.artist = song.artist || 'Unknown Artist';
      metadata.images = song.thumbnailUrl
        ? [new cast.framework.messages.Image(song.thumbnailUrl)]
        : [];

      mediaInfo.metadata = metadata;
      mediaInfo.contentId = song.id; // Or sourceId
      mediaInfo.contentType = 'audio/mpeg'; // Generic content type
      mediaInfo.streamType = cast.framework.messages.StreamType.BUFFERED;
      mediaInfo.duration = song.duration || 0;

      playerManager.setMediaInformation(mediaInfo);

      // Force a status broadcast
      (playerManager as ExtendedPlayerManager).broadcastStatus?.(true);
    });

    if (err) {
      // Logging removed as requested
    }
  }, []);

  return updateMediaMetadata;
}
