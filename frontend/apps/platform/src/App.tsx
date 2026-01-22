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
  console.warn(
    '🔥 [App] initialData.createRoomName:',
    initialData?.createRoomName,
  );

  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route
        path="/rooms/create"
        element={<CreateRoom initialData={initialData} />}
      />
      <Route
        path="/rooms/:id"
        element={<RoomView initialData={initialData} />}
      />
      <Route path="/player-test" element={<PlayerTest />} />
      <Route path="/callback" element={<Callback />} />
    </Routes>
  );
}
