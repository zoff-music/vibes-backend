import { Song } from '@vibez/shared';
import { create } from 'zustand';

interface QueueState {
  songs: Song[];

  setSongs: (songs: Song[]) => void;
  addSong: (song: Song) => void;
  removeSong: (songId: string) => void;
  updateSong: (song: Song) => void;
}

export const useQueueStore = create<QueueState>((set) => ({
  songs: [],

  setSongs: (songs) => set({ songs: [...songs] }), // Songs are already sorted by backend (vote_count DESC, added_at ASC)

  addSong: (song) =>
    set((state) => {
      if (state.songs.some((s) => s.id === song.id)) {
        return state;
      }
      return {
        songs: [...state.songs, song], // Backend handles sorting
      };
    }),

  removeSong: (songId) =>
    set((state) => ({
      songs: state.songs.filter((s) => s.id !== songId),
    })),

  updateSong: (song) =>
    set((state) => ({
      songs: state.songs.map((s) => (s.id === song.id ? song : s)),
    })),
}));
