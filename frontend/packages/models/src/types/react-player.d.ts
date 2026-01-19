declare module 'react-player' {
  import * as React from 'react';

  export interface ReactPlayerProps {
    url: string | string[] | React.ReactElement;
    playing?: boolean;
    controls?: boolean;
    width?: string | number;
    height?: string | number;
    onReady?: () => void;
    onEnded?: () => void;
    onError?: (error: unknown) => void;
    onStart?: () => void;
    onPlay?: () => void;
    onPause?: () => void;
    config?: {
      youtube?: {
        playerVars?: {
          modestbranding?: 0 | 1;
          controls?: 0 | 1;
          rel?: 0 | 1;
          autoplay?: 0 | 1;
          enablejsapi?: 0 | 1;
          origin?: string;
          playsinline?: 0 | 1;
        };
      };
    };
    style?: React.CSSProperties;
    pip?: boolean;
    stopOnUnmount?: boolean;
    light?: boolean;
  }

  export default class ReactPlayer extends React.Component<ReactPlayerProps> {
    getCurrentTime(): number;
    seekTo(amount: number, type?: 'seconds' | 'fraction'): void;
  }
}
