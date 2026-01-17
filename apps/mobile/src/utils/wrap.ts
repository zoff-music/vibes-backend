/**
 * Error wrapper utilities - use instead of try/catch
 * Per AGENTS.md: Do not use try/catch. Use safeWrap/safeWrapAsync for error capture.
 */

type Result<T, E = Error> = [E, null] | [null, T];

/**
 * Wraps a synchronous function call and returns a tuple [error, result]
 */
export function safeWrap<T>(fn: () => T): Result<T> {
  try {
    const result = fn();
    return [null, result];
  } catch (error) {
    return [error instanceof Error ? error : new Error(String(error)), null];
  }
}

/**
 * Wraps an async function call and returns a tuple [error, result]
 */
export async function safeWrapAsync<T>(
  fn: () => Promise<T>
): Promise<Result<T>> {
  try {
    const result = await fn();
    return [null, result];
  } catch (error) {
    return [error instanceof Error ? error : new Error(String(error)), null];
  }
}

/**
 * Wraps a promise and returns a tuple [error, result]
 */
export async function safeAwait<T>(promise: Promise<T>): Promise<Result<T>> {
  try {
    const result = await promise;
    return [null, result];
  } catch (error) {
    return [error instanceof Error ? error : new Error(String(error)), null];
  }
}

/**
 * Type guard to check if result is an error
 */
export function isError<T, E = Error>(
  result: Result<T, E>
): result is [E, null] {
  return result[0] !== null;
}

/**
 * Type guard to check if result is successful
 */
export function isSuccess<T, E = Error>(
  result: Result<T, E>
): result is [null, T] {
  return result[0] === null;
}
