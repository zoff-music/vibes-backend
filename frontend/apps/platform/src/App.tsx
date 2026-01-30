import type {
  AdminRoomSummary,
  PlaybackState,
  Room as RoomModel,
  Song,
} from '@vibez/models';
import { DebugConsole } from '@vibez/ui';
import { Route, Routes } from 'react-router';
import Admin from './pages/Admin';
import Callback from './pages/Callback';
import CreateRoom from './pages/CreateRoom';
import Home from './pages/Home';
import Room from './pages/Room';

export interface SSRInitialData {
  createRoomName?: string;
  room?: RoomModel;
  songs?: Song[];
  playback?: PlaybackState;
  theme?: 'dark' | 'light';
  adminRooms?: AdminRoomSummary[];
  adminAuthorized?: boolean;
}

interface AppProps {
  initialData?: SSRInitialData;
}

import { Background } from './components/layout/Background';

export default function App({ initialData }: AppProps) {
  console.warn('🔥 [App] Received initialData:', initialData);
  console.warn('🔥 [App] typeof window:', typeof window);
  console.warn(
    '🔥 [App] initialData.createRoomName:',
    initialData?.createRoomName,
  );

  return (
    <>
      <DebugConsole />
      <Background />
      <Routes>
        <Route path="/" element={<Home />} />
        <Route
          path="/rooms/create"
          element={<CreateRoom initialData={initialData} />}
        />
        <Route path="/rooms/:id" element={<Room initialData={initialData} />} />
        <Route path="/callback" element={<Callback />} />
        <Route path="/admin" element={<Admin initialData={initialData} />} />
      </Routes>
    </>
  );
}
