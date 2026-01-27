import { DebugConsole } from '@vibez/ui';
import { ActiveView } from './components/ActiveView';
import { CastProvider, useCast } from './components/CastProvider';
import { IdleView } from './components/IdleView';

const CastAppContent = () => {
  const { currentSong, debugMode } = useCast();

  return (
    <div className="relative flex h-screen w-screen animate-fade-in items-center justify-center overflow-hidden bg-theme text-theme">
      <div className="synth-sky absolute inset-0" />
      <div className="vhs-scanlines pointer-events-none absolute inset-0" />
      <div className="sun-hero opacity-80" />
      <div className="retro-grid opacity-70" />

      <div className="relative z-10 flex h-full w-full items-center justify-center">
        {currentSong ? <ActiveView /> : <IdleView />}
      </div>
      <DebugConsole enabled={debugMode} />
    </div>
  );
};

const App = () => {
  return (
    <CastProvider>
      <CastAppContent />
    </CastProvider>
  );
};

export default App;
