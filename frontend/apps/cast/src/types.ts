import type { Song } from '@vibez/shared';

export interface RoomInfo {
  name: string;
  participantCount: number;
}

export type QueueItem = Song;

export type LocalCastMessage =
  | {
      action: 'updatePlayback';
      currentSong?: QueueItem;
      isPlaying?: boolean;
      positionMs?: number;
      queue?: QueueItem[];
      roomInfo?: RoomInfo;
    }
  | {
      action: 'syncPlayback';
      currentSong?: QueueItem;
      isPlaying?: boolean;
      positionMs?: number;
    }
  | {
      action: 'updateQueue';
      queue?: QueueItem[];
    }
  | {
      action: 'updateRoomInfo';
      roomInfo?: RoomInfo;
    }
  | {
      action: 'joinRoom';
      roomId: string;
    };
