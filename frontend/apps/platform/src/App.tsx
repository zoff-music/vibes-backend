import { Route, Routes } from 'react-router';
import { safeWrap } from '@vibez/shared';
import Callback from './pages/Callback';
import CreateRoom from './pages/CreateRoom';
import Home from './pages/Home';
import PlayerTest from './pages/PlayerTest';
import RoomView from './pages/RoomView';

interface AppProps {
  initialData?: any;
}

export default function App({ initialData }: AppProps) {
  return (
    <html lang="en">
      <head>
        <meta charSet="UTF-8" />
        <link rel="icon" type="image/png" href="/favicon.png" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>Vibez - Shared Music Queue</title>
        <link rel="stylesheet" href="/index.css" />
        <script
          id="ssr-data"
          type="application/json"
          dangerouslySetInnerHTML={{
            __html: safeWrap(() => JSON.stringify(initialData))[1] || '{}'
          }}
        />
      </head>
      <body>
        <div id="root">
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/room/create" element={<CreateRoom />} />
            <Route path="/room/:id" element={<RoomView initialData={initialData?.room} />} />
            <Route path="/player-test" element={<PlayerTest />} />
            <Route path="/callback" element={<Callback />} />
          </Routes>
        </div>
      </body>
    </html>
  );
}
