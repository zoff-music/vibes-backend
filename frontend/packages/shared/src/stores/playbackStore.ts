import { type PlaybackState } from '@vibez/models';
import { create } from 'zustand';

interface PlaybackStoreState extends PlaybackState {
  // Client-side computed fields
  actualPositionMs: number;
  clientReferenceTime: number;

  // Server mode local state
  localIsPlaying: boolean | null; // null means use server state, boolean means local override
  roomMode: string | null;

  // Interval management
  autoUpdateInterval: ReturnType<typeof setInterval> | null;
  startAutoUpdate: () => void;
  stopAutoUpdate: () => void;

  setPlaybackState: (state: PlaybackState, roomMode?: string) => void;
  setIsPlaying: (isPlaying: boolean) => void;
  setLocalPlayingState: (isPlaying: boolean, roomMode: string) => void;
  updateActualPosition: () => void;
}

export const usePlaybackStore = create<PlaybackStoreState>((set, get) => ({
  currentSong: null,
  isPlaying: false,
  positionMs: 0,
  updatedAt: new Date().toISOString(),
  serverTimeMs: Date.now(),
  actualPositionMs: 0,
  clientReferenceTime: Date.now(),
  autoUpdateInterval: null,
  localIsPlaying: null,
  roomMode: null,

  setPlaybackState: (state, roomMode) => {
    const currentState = get();

    // In Server mode, preserve local playing state unless it's a new song
    if (roomMode === 'server' && currentState.localIsPlaying !== null) {
      const isNewSong =
        !currentState.currentSong ||
        !state.currentSong ||
        currentState.currentSong.id !== state.currentSong.id;

      if (!isNewSong) {
        // Keep local playing state, but update position and other fields
        set({
          ...state,
          isPlaying: currentState.localIsPlaying,
          clientReferenceTime: Date.now(),
          roomMode: roomMode || currentState.roomMode,
        });
        get().updateActualPosition();
        return;
      } else {
        // New song - clear local override
        set({
          ...state,
          clientReferenceTime: Date.now(),
          localIsPlaying: null,
          roomMode: roomMode || currentState.roomMode,
        });
        get().updateActualPosition();
        return;
      }
    }

    // Host mode or no local override - use server state
    set({
      ...state,
      clientReferenceTime: Date.now(),
      localIsPlaying: null,
      roomMode: roomMode || currentState.roomMode,
    });
    get().updateActualPosition();

    // Manage update interval based on playback state
    if (state.isPlaying) {
      get().startAutoUpdate();
    } else {
      get().stopAutoUpdate();
    }
  },

  setIsPlaying: (isPlaying) => {
    set({ isPlaying });
    if (isPlaying) {
      get().startAutoUpdate();
    } else {
      get().stopAutoUpdate();
    }
  },

  setLocalPlayingState: (isPlaying, roomMode) => {
    if (roomMode === 'server') {
      set({
        isPlaying,
        localIsPlaying: isPlaying,
        roomMode,
      });
    } else {
      // Host mode - just set normally
      set({
        isPlaying,
        localIsPlaying: null,
        roomMode,
      });
    }

    if (isPlaying) {
      get().startAutoUpdate();
    } else {
      get().stopAutoUpdate();
    }
  },

  updateActualPosition: () => {
    const { positionMs, isPlaying, clientReferenceTime, currentSong } = get();

    if (!isPlaying) {
      set({ actualPositionMs: positionMs });
      return;
    }

    // Simple calculation: add time elapsed since we received this state
    const elapsedOnClient = Math.max(0, Date.now() - clientReferenceTime);
    let newPositionMs = positionMs + elapsedOnClient;

    // Clamp to song duration if available
    if (currentSong?.duration) {
      const durationMs = currentSong.duration * 1000;
      newPositionMs = Math.min(newPositionMs, durationMs);
    }

    set({ actualPositionMs: newPositionMs });
  },

  startAutoUpdate: () => {
    const { autoUpdateInterval, isPlaying } = get();
    if (autoUpdateInterval || !isPlaying) return;

    const interval = setInterval(() => {
      get().updateActualPosition();
    }, 100);

    set({ autoUpdateInterval: interval });
  },

  stopAutoUpdate: () => {
    const { autoUpdateInterval } = get();
    if (!autoUpdateInterval) return;

    clearInterval(autoUpdateInterval);
    set({ autoUpdateInterval: null });
  },
}));
