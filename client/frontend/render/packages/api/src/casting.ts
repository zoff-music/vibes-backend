import type { CastingTokenResponse } from '@vibez/models';
import { api } from './index';

export async function createCastingToken(
  roomId: string,
): Promise<[Error | null, CastingTokenResponse | null]> {
  return api.post('/casting/tokens', null, { roomId });
}
