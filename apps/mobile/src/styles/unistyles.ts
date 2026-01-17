import { UnistylesRegistry } from 'react-native-unistyles';
import { darkTheme, breakpoints } from './theme';

type AppBreakpoints = typeof breakpoints;
type AppThemes = {
  dark: typeof darkTheme;
};

declare module 'react-native-unistyles' {
  export interface UnistylesBreakpoints extends AppBreakpoints {}
  export interface UnistylesThemes extends AppThemes {}
}

UnistylesRegistry.addBreakpoints(breakpoints)
  .addThemes({
    dark: darkTheme,
  })
  .addConfig({
    adaptiveThemes: false,
    initialTheme: 'dark',
  });
