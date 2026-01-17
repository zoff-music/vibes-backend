/**
 * Safe wrapper for synchronous functions.
 * Returns [data, error] tuple.
 */
export function safeWrap<T>(fn: () => T): [T | null, Error | null] {
  try {
    return [fn(), null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}

/**
 * Safe wrapper for asynchronous functions.
 * Returns [data, error] tuple.
 */
export async function safeWrapAsync<T>(promise: Promise<T>): Promise<[T | null, Error | null]> {
  try {
    const data = await promise;
    return [data, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}
