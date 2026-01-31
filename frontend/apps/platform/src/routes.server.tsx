import { redirect, type RouteObject } from 'react-router';
import App, { type SSRInitialData } from './App';
import Admin from './pages/Admin';
import Callback from './pages/Callback';
import CreateRoom from './pages/CreateRoom';
import Home from './pages/Home';
import Room from './pages/Room';

export function createServerRoutes(
  initialData?: SSRInitialData,
): RouteObject[] {
  return [
    {
      path: '/',
      element: <App initialData={initialData} />,
      children: [
        { index: true, element: <Home /> },
        { path: 'rooms/create', element: <CreateRoom /> },
        { path: 'rooms/:id', element: <Room /> },
        { path: 'callback', element: <Callback /> },
        { path: 'admin', element: <Admin /> },
        { path: '*', loader: () => redirect('/') },
      ],
    },
  ];
}
