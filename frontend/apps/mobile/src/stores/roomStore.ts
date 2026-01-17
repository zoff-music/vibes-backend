import { create } from 'zustand';
import { Room, RoomUser } from '@vibez/shared';

interface RoomState {
  room: Room | null;
  users: RoomUser[];
  userId: string | null;
  isAdmin: boolean;
  nickname: string | null;
  
  setRoom: (room: Room) => void;
  setUsers: (users: RoomUser[]) => void;
  setSession: (userId: string, isAdmin: boolean, nickname?: string) => void;
  reset: () => void;
}

export const useRoomStore = create<RoomState>((set) => ({
  room: null,
  users: [],
  userId: null,
  isAdmin: false,
  nickname: null,

  setRoom: (room) => set({ room }),
  setUsers: (users) => set({ users }),
  setSession: (userId, isAdmin, nickname) => set({ 
    userId, 
    isAdmin, 
    nickname: nickname || null 
  }),
  reset: () => set({ 
    room: null, 
    users: [], 
    userId: null, 
    isAdmin: false, 
    nickname: null 
  }),
}));
