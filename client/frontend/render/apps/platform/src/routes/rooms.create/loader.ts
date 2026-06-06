import type { LoaderFunctionArgs } from 'react-router';

export interface RoomsCreateLoaderData {
  createRoomName?: string;
}

export async function loader({
  request,
}: LoaderFunctionArgs): Promise<RoomsCreateLoaderData> {
  const url = new URL(request.url);
  const name = url.searchParams.get('name') ?? undefined;
  return { createRoomName: name };
}
