import { create } from 'zustand';
import { PlaybackState } from '@vibez/shared';

interface PlaybackStoreState extends PlaybackState {
  // Client-side computed fields
  actualPositionMs: number;
  
  setPlaybackState: (state: PlaybackState) => void;
  setIsPlaying: (isPlaying: boolean) => void;
  updateActualPosition: () => void;
}

export const usePlaybackStore = create<PlaybackStoreState>((set, get) => ({
  currentSongId: null,
  currentSong: null,
  isPlaying: false,
  positionMs: 0,
  updatedAt: new Date().toISOString(),
  serverTimeMs: Date.now(),
  actualPositionMs: 0,

  setPlaybackState: (state) => {
    set({ ...state });
    get().updateActualPosition();
  },

  setIsPlaying: (isPlaying) => set({ isPlaying }),

  updateActualPosition: () => {
    const { positionMs, updatedAt, serverTimeMs, isPlaying } = get();
    
    if (!isPlaying) {
      set({ actualPositionMs: positionMs });
      return;
    }

    // Calculate drift: how much time has passed since the state was last updated
    // We use serverTimeMs as the reference for when updatedAt was captured
    const now = Date.now();
    const timeSinceUpdate = now - serverTimeMs;
    
    set({ actualPositionMs: positionMs + timeSinceUpdate });
  },
}));
