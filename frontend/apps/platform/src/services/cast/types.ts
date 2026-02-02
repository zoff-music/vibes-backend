// Cast message types for communication between sender and receiver

export type LocalCastMessage =
  | {
      action: 'receiverReady';
      timestamp: number;
    }
  | {
      action: 'updatePlayback';
      currentSong: {
        id: string;
        title: string;
        artist: string;
        sourceType: string;
        sourceId: string;
        thumbnailUrl?: string;
        duration?: number;
      };
      isPlaying: boolean;
      positionMs: number;
      queue: Array<{
        id: string;
        title: string;
        artist: string;
        sourceType: string;
        sourceId: string;
        thumbnailUrl?: string;
        duration?: number;
      }>;
      roomInfo: {
        name: string;
        participantCount: number;
      };
      timestamp: number;
    }
  | {
      action: 'joinRoom';
      roomId: string;
      casterId?: string;
      sessionId?: string;
      timestamp: number;
    }
  | {
      action: 'updateQueue';
      queue: Array<{
        id: string;
        title: string;
        artist: string;
        sourceType: string;
        sourceId: string;
        thumbnailUrl?: string;
        duration?: number;
      }>;
      timestamp: number;
    }
  | {
      action: 'updateRoomInfo';
      roomInfo: {
        name: string;
        participantCount: number;
      };
      timestamp: number;
    }
  | {
      action: 'syncPlayback';
      isPlaying: boolean;
      positionMs: number;
      currentSong?: {
        id: string;
        title: string;
        artist: string;
        sourceType: string;
        sourceId: string;
        thumbnailUrl?: string;
        duration?: number;
      };
      timestamp: number;
    };
