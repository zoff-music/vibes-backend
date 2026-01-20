import { type PlaybackState } from '@vibez/models';
import { create } from 'zustand';

interface PlaybackStoreState extends PlaybackState {
  // Client-side computed fields
  actualPositionMs: number;
  clientReferenceTime: number;

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
  clientReferenceTime: Date.now(),

  setPlaybackState: (state) => {
    set({
      ...state,
      clientReferenceTime: Date.now(),
    });
    get().updateActualPosition();
  },

  setIsPlaying: (isPlaying) => set({ isPlaying }),

  updateActualPosition: () => {
    const {
      positionMs,
      updatedAt,
      serverTimeMs,
      isPlaying,
      clientReferenceTime,
    } = get();

    if (!isPlaying) {
      set({ actualPositionMs: positionMs });
      return;
    }

    // Calculate drift:
    // 1. Calculate how much time passed on server before it sent the state
    //    elapsedOnServer = serverTimeMs - updatedAt
    // 2. Calculate how much time passed on client since we received the state
    //    elapsedOnClient = Date.now() - clientReferenceTime

    const serverUpdatedAt = new Date(updatedAt).getTime();
    const elapsedOnServer = Math.max(0, serverTimeMs - serverUpdatedAt);
    const elapsedOnClient = Math.max(0, Date.now() - clientReferenceTime);

    set({ actualPositionMs: positionMs + elapsedOnServer + elapsedOnClient });
  },
}));
