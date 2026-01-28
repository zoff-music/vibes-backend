import { Room } from '@vibez/models';
import { create } from 'zustand';
import { RoomUser } from '../types';

interface RoomState {
  room: Room | null;
  users: RoomUser[];
  userId: string | null;
  isAdmin: boolean;
  nickname: string | null;
  usersCount: number;

  setRoom: (room: Room) => void;
  setUsers: (users: RoomUser[]) => void;
  setUsersCount: (count: number) => void;
  setSession: (userId: string, isAdmin: boolean, nickname?: string) => void;
  reset: () => void;
}

export const useRoomStore = create<RoomState>((set) => ({
  room: null,
  users: [],
  userId: null,
  isAdmin: false,
  nickname: null,
  usersCount: 0,

  setRoom: (room) => {
    const isAdmin = room.isAdmin !== undefined ? room.isAdmin : false;
    const usersCount = room.userCount !== undefined ? room.userCount : 0;
    const userId = (room as any).userId;

    set((state) => ({
      room,
      isAdmin,
      usersCount,
      userId: userId || state.userId,
    }));
  },
  setUsers: (users) => set({ users, usersCount: users.length }),
  setUsersCount: (usersCount) => set({ usersCount }),
  setSession: (userId, isAdmin, nickname) =>
    set({
      userId,
      isAdmin,
      nickname: nickname || null,
    }),
  reset: () =>
    set({
      room: null,
      users: [],
      userId: null,
      isAdmin: false,
      nickname: null,
      usersCount: 0,
    }),
}));
