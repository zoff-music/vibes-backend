import { Song } from '@vibez/shared';
import { create } from 'zustand';

interface QueueState {
  songs: Song[];

  setSongs: (songs: Song[]) => void;
  addSong: (song: Song) => void;
  removeSong: (songId: string) => void;
  updateSong: (song: Song) => void;
  reorderSongs: (songId: string, newPosition: number) => void;
}

export const useQueueStore = create<QueueState>((set) => ({
  songs: [],

  setSongs: (songs) =>
    set({ songs: [...songs].sort((a, b) => a.position - b.position) }),

  addSong: (song) =>
    set((state) => {
      if (state.songs.some((s) => s.id === song.id)) {
        return state;
      }
      return {
        songs: [...state.songs, song].sort((a, b) => a.position - b.position),
      };
    }),

  removeSong: (songId) =>
    set((state) => ({
      songs: state.songs.filter((s) => s.id !== songId),
    })),

  updateSong: (song) =>
    set((state) => ({
      songs: state.songs
        .map((s) => (s.id === song.id ? song : s))
        .sort((a, b) => a.position - b.position),
    })),

  reorderSongs: (songId, newPosition) =>
    set((state) => {
      // Basic local reorder for immediate feedback
      const newSongs = [...state.songs];
      const index = newSongs.findIndex((s) => s.id === songId);
      if (index === -1) return state;

      const [song] = newSongs.splice(index, 1);
      song.position = newPosition;
      newSongs.splice(newPosition, 0, song);

      // Refresh all positions to be safe
      return {
        songs: newSongs.map((s, i) => ({ ...s, position: i })),
      };
    }),
}));
