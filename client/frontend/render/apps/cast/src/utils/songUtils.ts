import type { Song } from '@vibez/shared';

export const normalizeSong = (song: Song): Song => {
  return {
    id: String(song.id), // Ensure ID is always a string
    sourceType: song.sourceType,
    sourceId: song.sourceId,
    title: song.title,
    artist: song.artist,
    thumbnailUrl: song.thumbnailUrl || '',
    duration: song.duration,
    voteCount: song.voteCount,
    addedBy: song.addedBy,
    addedAt: song.addedAt || new Date().toISOString(),
  };
};
