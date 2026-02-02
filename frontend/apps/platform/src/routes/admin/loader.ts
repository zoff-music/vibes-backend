import type { AdminRoomSummary } from '@vibez/models';
import type { LoaderFunctionArgs } from 'react-router';
import { getServerApi } from '../../http.server';

export interface AdminLoaderData {
  adminRooms: AdminRoomSummary[];
  adminAuthorized: boolean;
}

export async function loader({
  request,
}: LoaderFunctionArgs): Promise<AdminLoaderData> {
  const serverApi = getServerApi(request);
  const cookieHeader = request.headers.get('cookie') ?? undefined;
  const requestHeaders = cookieHeader ? { Cookie: cookieHeader } : undefined;

  const [roomsErr, rooms] = await serverApi.get('/admin/rooms', null, {
    headers: requestHeaders,
  });

  return {
    adminRooms: roomsErr ? [] : rooms || [],
    adminAuthorized: !roomsErr,
  };
}
