import { index, route } from '@react-router/dev/routes';

export default [
  index('./routes/_index/route.tsx'),
  route('rooms/create', './routes/rooms.create/route.tsx'),
  route('rooms/:id', './routes/rooms.$id/route.tsx'),
  route('callback', './routes/callback/route.tsx'),
  route('admin', './routes/admin/route.tsx'),
  route(':id', './routes/rooms.alias/route.tsx'),
];
