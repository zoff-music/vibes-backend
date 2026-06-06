/**
 * Safe wrapper for synchronous functions.
 * Returns [error, data] tuple.
 */
export function safeWrap<T>(fn: () => T): [Error | null, T | null] {
  try {
    return [null, fn()];
  } catch (err) {
    return [err instanceof Error ? err : new Error(String(err)), null];
  }
}

/**
 * Safe wrapper for asynchronous functions.
 * Returns [error, data] tuple.
 */
export async function safeWrapAsync<T>(
  promise: Promise<T>,
): Promise<[Error | null, T | null]> {
  try {
    const data = await promise;
    return [null, data];
  } catch (err) {
    return [err instanceof Error ? err : new Error(String(err)), null];
  }
}
