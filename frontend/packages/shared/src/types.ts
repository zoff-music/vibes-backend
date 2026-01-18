// Room types
export interface Room {
  id: string;
  name: string;
  createdAt: string;
  hasPassword: boolean;
  settings: RoomSettings;
  userCount?: number;
}

export interface RoomSettings {
  skipAllowed: boolean;
  democraticSkip: boolean;
  skipVoteThreshold: number;
  maxContinuousAdds: number;
  removeOnPlay: boolean;
  loopQueue: boolean;
  allowDuplicates: boolean;
}

export const DEFAULT_ROOM_SETTINGS: RoomSettings = {
  skipAllowed: true,
  democraticSkip: false,
  skipVoteThreshold: 0.5,
  maxContinuousAdds: 3,
  removeOnPlay: true,
  loopQueue: false,
  allowDuplicates: false,
};

// Song types
export type SourceType = 'youtube' | 'spotify' | 'soundcloud';

export interface Song {
  id: string;
  sourceType: SourceType;
  sourceId: string;
  title: string;
  artist?: string;
  thumbnailUrl: string;
  duration: number;
  addedBy: string;
  addedByNickname?: string;
  addedAt: string;
  position: number;
}

// Playback types
export interface PlaybackState {
  currentSongId: string | null;
  currentSong: Song | null;
  isPlaying: boolean;
  positionMs: number;
  updatedAt: string;
  serverTimeMs: number;
}

// User types
export interface RoomUser {
  id: string;
  nickname?: string;
  isAdmin: boolean;
  joinedAt: string;
  lastSeenAt: string;
}

// Session types
export interface Session {
  userId: string;
  nickname?: string;
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

export interface UsersUpdateEvent extends SSEEvent<RoomUser[]> {
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
