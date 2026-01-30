let currentPath: string | null = null;
let previousPath: string | null = null;

export const updateNavigationHistory = (nextPath: string) => {
  if (currentPath === nextPath) return;
  previousPath = currentPath;
  currentPath = nextPath;
};

export const getPreviousPath = () => previousPath;
export const getCurrentPath = () => currentPath;
