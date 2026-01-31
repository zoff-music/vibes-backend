import { createContext, type ReactNode, useContext } from 'react';
import type { SSRInitialData } from '../App';

interface InitialDataProviderProps {
  initialData?: SSRInitialData;
  children: ReactNode;
}

const InitialDataContext = createContext<SSRInitialData | undefined>(undefined);

export function InitialDataProvider({
  initialData,
  children,
}: InitialDataProviderProps) {
  return (
    <InitialDataContext.Provider value={initialData}>
      {children}
    </InitialDataContext.Provider>
  );
}

export function useInitialData() {
  return useContext(InitialDataContext);
}
