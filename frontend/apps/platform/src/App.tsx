import { safeWrap } from '@vibez/shared';
import { Route, Routes } from 'react-router';
import Callback from './pages/Callback';
import CreateRoom from './pages/CreateRoom';
import Home from './pages/Home';
import PlayerTest from './pages/PlayerTest';
import RoomView from './pages/RoomView';

interface AppProps {
  initialData?: any;
}

export default function App({ initialData }: AppProps) {
  console.warn('🔥 [App] Received initialData:', initialData);
  console.warn('🔥 [App] typeof window:', typeof window);
  console.warn('🔥 [App] initialData.createRoomName:', initialData?.createRoomName);
  
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/room/create" element={<CreateRoom initialData={initialData} />} />
      <Route
        path="/room/:id"
        element={<RoomView initialData={initialData?.room} />}
      />
      <Route path="/player-test" element={<PlayerTest />} />
      <Route path="/callback" element={<Callback />} />
    </Routes>
  );
}
