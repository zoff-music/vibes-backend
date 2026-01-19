import { Routes, Route } from 'react-router-dom';
import Home from './pages/Home';
import CreateRoom from './pages/CreateRoom';
import RoomView from './pages/RoomView';
import PlayerTest from './pages/PlayerTest';

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/room/create" element={<CreateRoom />} />
      <Route path="/room/:id" element={<RoomView />} />
      <Route path="/player-test" element={<PlayerTest />} />
    </Routes>
  );
}
