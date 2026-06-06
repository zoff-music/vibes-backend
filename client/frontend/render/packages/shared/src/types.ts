import {
  PlaybackState,
  Room,
  RoomSettings,
  RoomUpdate,
  Song,
  SourceType,
} from '@vibez/models';

export type { Room, RoomSettings, Song, SourceType, PlaybackState, RoomUpdate };

export const DEFAULT_ROOM_SETTINGS: RoomSettings = {
  skipAllowed: true,
  democraticSkip: false,
  skipVoteThreshold: 0.5,
  maxContinuousAdds: 3,
  removeOnPlay: true,
  loopQueue: false,
  allowDuplicates: false,
  enabledSources: ['youtube', 'spotify', 'soundcloud'],
};

// User types
export interface RoomUser {
  id: string;
  isAdmin: boolean;
  joinedAt: string;
  lastSeenAt: string;
}

// Session types
export interface Session {
  userId: string;
  isAdmin: boolean;
  room: Room;
}

// Action types
export type RoomAction = 'play' | 'pause' | 'seek' | 'skip' | 'vote';

export interface RoomActionRequest {
  action: RoomAction;
  positionMs?: number;
}

export interface PlayActionResponse {
  action: 'play';
  playback: PlaybackState;
}

export interface PauseActionResponse {
  action: 'pause';
  playback: PlaybackState;
}

export interface SeekActionResponse {
  action: 'seek';
  playback: PlaybackState;
}

export interface SkipActionResponse {
  action: 'skip';
  skipped: boolean;
  nextSong: Song | null;
  playback: PlaybackState;
}

export interface VoteActionResponse {
  action: 'vote';
  voted: boolean;
  currentVotes: number;
  requiredVotes: number;
  skipped: boolean;
  nextSong: Song | null;
  playback?: PlaybackState;
}

export type RoomActionResponse =
  | PlayActionResponse
  | PauseActionResponse
  | SeekActionResponse
  | SkipActionResponse
  | VoteActionResponse;

// SSE Event types
export type SSEEventType =
  | 'room_state'
  | 'songs_update'
  | 'playback_update'
  | 'users_update'
  | 'song_added'
  | 'skip_vote';

export interface SSEEvent<T = unknown> {
  type: SSEEventType;
  data: T;
}

export interface RoomStateEvent extends SSEEvent<Room> {
  type: 'room_state';
}

export interface SongsUpdateEvent extends SSEEvent<Song[]> {
  type: 'songs_update';
}

export interface PlaybackUpdateEvent extends SSEEvent<PlaybackState> {
  type: 'playback_update';
}

export interface UsersUpdateEvent extends SSEEvent<number> {
  type: 'users_update';
}

export interface SkipVoteEvent
  extends SSEEvent<{ userId: string; songId: string }> {
  type: 'skip_vote';
}

export interface SongAddedEvent extends SSEEvent<Song> {
  type: 'song_added';
}

// API Error types
export interface APIError {
  error: string;
  message: string;
  details?: Record<string, unknown>;
}
