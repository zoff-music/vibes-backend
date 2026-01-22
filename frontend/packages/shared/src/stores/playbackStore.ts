import { type PlaybackState } from '@vibez/models';
import { create } from 'zustand';

interface PlaybackStoreState extends PlaybackState {
  // Client-side computed fields
  actualPositionMs: number;
  clientReferenceTime: number;
  
  // Server mode local state
  localIsPlaying: boolean | null; // null means use server state, boolean means local override
  roomMode: string | null;

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
  localIsPlaying: null,
  roomMode: null,

  setPlaybackState: (state, roomMode) => {
    const currentState = get();
    
    // In Server mode, preserve local playing state unless it's a new song
    if (roomMode === 'server' && currentState.localIsPlaying !== null) {
      const isNewSong = !currentState.currentSong || 
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
  },

  setIsPlaying: (isPlaying) => set({ isPlaying }),

  setLocalPlayingState: (isPlaying, roomMode) => {
    if (roomMode === 'server') {
      set({ 
        isPlaying,
        localIsPlaying: isPlaying,
        roomMode 
      });
    } else {
      // Host mode - just set normally
      set({ 
        isPlaying,
        localIsPlaying: null,
        roomMode 
      });
    }
  },

  updateActualPosition: () => {
    const {
      positionMs,
      isPlaying,
      clientReferenceTime,
    } = get();

    if (!isPlaying) {
      set({ actualPositionMs: positionMs });
      return;
    }

    // Simple calculation: add time elapsed since we received this state
    const elapsedOnClient = Math.max(0, Date.now() - clientReferenceTime);
    set({ actualPositionMs: positionMs + elapsedOnClient });
  },
}));