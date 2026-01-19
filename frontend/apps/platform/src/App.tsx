import { Route, Routes } from 'react-router-dom';
import CreateRoom from './pages/CreateRoom';
import Home from './pages/Home';
import PlayerTest from './pages/PlayerTest';
import RoomView from './pages/RoomView';
import Callback from './pages/Callback';

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/room/create" element={<CreateRoom />} />
      <Route path="/room/:id" element={<RoomView />} />
      <Route path="/player-test" element={<PlayerTest />} />
      <Route path="/callback" element={<Callback />} />
    </Routes>
  );
}
