import { safeWrap } from '@vibez/shared';

export function getThemeFromCookies(
  cookieHeader: string | null,
): 'light' | 'dark' | 'auto' {
  if (!cookieHeader) {
    return 'auto';
  }

  const [cookieErr, cookies] = safeWrap(() =>
    cookieHeader.split(';').reduce<Record<string, string>>((acc, cookie) => {
      const trimmed = cookie.trim();
      if (!trimmed) return acc;
      const separatorIndex = trimmed.indexOf('=');
      if (separatorIndex === -1) return acc;
      const name = trimmed.slice(0, separatorIndex);
      const value = trimmed.slice(separatorIndex + 1);
      if (!name || !value) return acc;
      acc[name] = decodeURIComponent(value);
      return acc;
    }, {}),
  );

  if (cookieErr || !cookies) {
    if (cookieErr) {
      console.error('[SSR] Error parsing cookies:', cookieErr);
    }
    return 'auto';
  }

  let preferencesEncoded = cookies.preferences;
  if (!preferencesEncoded) {
    return 'auto';
  }

  if (preferencesEncoded.startsWith('"') && preferencesEncoded.endsWith('"')) {
    preferencesEncoded = preferencesEncoded.slice(1, -1);
  }

  const [preferencesErr, preferences] = safeWrap(() => {
    const normalized = preferencesEncoded.replace(/-/g, '+').replace(/_/g, '/');
    const padding = normalized.length % 4;
    const padded =
      padding === 0 ? normalized : normalized + '='.repeat(4 - padding);
    const preferencesJson = Buffer.from(padded, 'base64').toString('utf-8');
    return JSON.parse(preferencesJson) as { theme?: string };
  });

  if (preferencesErr || !preferences) {
    if (preferencesErr) {
      console.error('[SSR] Error parsing theme preferences:', preferencesErr);
    }
    return 'auto';
  }

  if (preferences.theme === 'dark') return 'dark';
  if (preferences.theme === 'auto') return 'auto';
  if (preferences.theme === 'light') return 'light';

  return 'auto';
}
