import React from 'react';
import { useCast } from './CastProvider';

export const IdleView: React.FC = () => {
  const { statusText, roomInfo } = useCast();

  return (
    <div className="relative z-10 flex max-w-3xl flex-col items-center gap-8 px-6 text-center">
      <div className="panel-frame panel-surface w-full px-10 py-12">
        <div className="mb-6 flex flex-col items-center gap-3">
          <span
            className="vhs-tear-strong glow-text font-display text-6xl text-readable text-theme uppercase tracking-[0.22em] md:text-7xl"
            data-text="ゾフ"
          >
            ゾフ
          </span>
          <p className="font-mono text-theme-subtle text-xs lowercase tracking-[0.4em]">
            zoff
          </p>
        </div>
        <p className="font-display text-2xl text-readable text-theme">
          {statusText}
        </p>
        <p className="mt-3 text-base text-theme-muted">
          Waiting for music to play...
        </p>
      </div>

      {roomInfo && (
        <div className="panel-frame panel-surface w-full px-8 py-6">
          <p className="font-display text-3xl text-theme">{roomInfo.name}</p>
          <div className="mt-3 flex items-center justify-center gap-2 text-theme-muted">
            <span className="h-2 w-2 animate-pulse rounded-full bg-[var(--color-secondary)]" />
            <span className="text-sm uppercase tracking-[0.25em]">
              {roomInfo.participantCount} active
            </span>
          </div>
        </div>
      )}
    </div>
  );
};
