const TRUTHY_VALUES = new Set(['true', '1', 'yes', 'on']);

export const isTruthyFlag = (value?: string | null): boolean => {
  if (!value) return false;
  return TRUTHY_VALUES.has(value.trim().toLowerCase());
};

export const applyConsoleLogGuard = (enabled: boolean): void => {
  if (enabled) return;
  const noop = (): void => {};
  console.log = noop;
  console.info = noop;
  console.debug = noop;
  console.warn = noop;
};
