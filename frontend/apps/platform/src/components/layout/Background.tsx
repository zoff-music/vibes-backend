import { useLocation } from 'react-router';
import { useThemeStore } from '../../stores/themeStore';

export function Background() {
  const location = useLocation();
  const isWarping = useThemeStore((state) => state.isWarping);

  // Only show the sun on Home and CreateRoom pages
  const showSun =
    location.pathname === '/' || location.pathname === '/rooms/create';

  return (
    <div className="pointer-events-none fixed inset-0 z-0">
      <div className="synth-sky absolute inset-0" />
      <div className="synth-haze absolute inset-0" />
      <div className="vhs-scanlines absolute inset-0" />

      {showSun && <div className="sun-hero sunset-orb absolute" />}

      <div
        className={`retro-grid absolute bottom-0 left-1/2 h-[100vh] w-[200%] ${
          isWarping ? 'grid-warp' : ''
        }`}
      />
    </div>
  );
}
