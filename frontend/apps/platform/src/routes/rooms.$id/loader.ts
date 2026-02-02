import type { PlaybackState } from '@vibez/shared';
import type { LoaderFunctionArgs } from 'react-router';
import { redirect } from 'react-router';
import type { Room as RoomModel, Song } from '@vibez/models';
import { getServerApi } from '../../http.server';

export interface RoomLoaderData {
  room: RoomModel;
  songs: Song[];
  playback?: PlaybackState;
}

export async function loader({
  request,
  params,
}: LoaderFunctionArgs): Promise<RoomLoaderData | Response> {
  const roomId = params.id;
  if (!roomId) {
    return redirect('/rooms/create');
  }

  console.log('[rooms.$id loader] start', {
    roomId,
    requestUrl: request.url,
    apiUrl: process?.env?.VITE_API_URL,
    apiUrlInternal: process?.env?.VITE_API_URL_INTERNAL,
  });

  const serverApi = getServerApi(request);
  const cookieHeader = request.headers.get('cookie') ?? undefined;
  const requestHeaders = cookieHeader ? { Cookie: cookieHeader } : undefined;

  const [roomRes, songsRes, playbackRes] = await Promise.all([
    serverApi.get('/rooms/{id}', { id: roomId }, { headers: requestHeaders }),
    serverApi.get('/rooms/{id}/songs', { id: roomId }, { headers: requestHeaders }),
    serverApi.get('/rooms/{id}/states', { id: roomId }, { headers: requestHeaders }),
  ]);

  const [roomErr, room] = roomRes;
  const [songsErr, songs] = songsRes;
  const [playbackErr, playback] = playbackRes;
  if (roomErr || songsErr || playbackErr) {
    console.warn('[rooms.$id loader] api errors', {
      roomErr,
      songsErr,
      playbackErr,
    });
  }
  if (roomErr || !room) {
    const createUrl = new URL('/rooms/create', request.url);
    createUrl.searchParams.set('name', roomId);
    return redirect(createUrl.toString());
  }

  return {
    room,
    songs: songs || [],
    playback: (playback || undefined) as PlaybackState | undefined,
  };
}
