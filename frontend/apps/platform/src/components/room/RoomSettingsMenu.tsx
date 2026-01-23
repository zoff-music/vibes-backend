import type { Room, RoomSettings, RoomUpdate } from '@vibez/models';
import { MoonIcon, ShareIcon, SunIcon } from '@vibez/ui';
import { AnimatePresence, motion } from 'framer-motion';
import { type RefObject, useEffect } from 'react';
import { RoomSharePanel } from './RoomSharePanel';

interface RoomSettingsMenuProps {
  showSettings: boolean;
  onClose: () => void;
  showShare: boolean;
  onToggleShare: () => void;
  isDarkMode: boolean;
  onToggleDarkMode: () => void;
  room: Room | null;
  displayRoom: Room | null;
  isAdmin: boolean;
  updateRoomSettings: (settings: RoomSettings) => void;
  updateRoom: (data: RoomUpdate) => Promise<Room | null> | null;
  adminPassword: string;
  onAdminPasswordChange: (value: string) => void;
  onJoinAdmin: () => void;
  isAuthenticating: boolean;
  shareUrl: string;
  onCopyShareLink: () => void;
  settingsMenuRef?: RefObject<HTMLDivElement | null>;
}

export const RoomSettingsMenu = ({
  showSettings,
  onClose,
  showShare,
  onToggleShare,
  isDarkMode,
  onToggleDarkMode,
  room,
  displayRoom,
  isAdmin,
  updateRoomSettings,
  updateRoom,
  adminPassword,
  onAdminPasswordChange,
  onJoinAdmin,
  isAuthenticating,
  shareUrl,
  onCopyShareLink,
  settingsMenuRef,
}: RoomSettingsMenuProps) => {
  useEffect(() => {
    if (!showSettings) return;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [showSettings]);

  return (
    <AnimatePresence>
      {showSettings && (
        <div className="fixed top-[var(--room-header-height)] right-0 bottom-0 left-0 z-40">
          <button
            type="button"
            className="absolute inset-0 h-full w-full cursor-pointer bg-transparent"
            onClick={onClose}
            aria-label="Close settings"
          />
          <motion.div
            ref={settingsMenuRef}
            initial={{ opacity: 0, y: 10, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 10, scale: 0.95 }}
            className="fixed top-[var(--room-header-height)] right-0 left-0 z-10 h-[calc(100dvh-var(--room-header-height))] w-full overflow-y-auto overscroll-contain border-theme border-t bg-theme-surface p-5 text-theme shadow-2xl sm:absolute sm:top-auto sm:right-0 sm:left-auto sm:mt-3 sm:h-auto sm:w-72 sm:overflow-visible sm:rounded-3xl sm:border"
          >
            <div className="space-y-4">
              <div className="space-y-3 sm:hidden">
                <div className="flex items-center gap-3">
                  <button
                    onClick={onToggleShare}
                    className={`flex flex-1 items-center justify-center gap-2 rounded-xl border p-3 text-xs transition-all ${
                      showShare
                        ? 'border-theme-strong bg-theme-surface text-theme'
                        : 'border-theme text-theme-muted hover:border-theme-strong hover:text-theme'
                    }`}
                    title="Share Room"
                  >
                    <ShareIcon className="h-4 w-4" />
                    Share
                  </button>

                  <button
                    onClick={onToggleDarkMode}
                    className={`flex flex-1 items-center justify-center gap-2 rounded-xl border p-3 text-xs transition-all ${
                      isDarkMode
                        ? 'border-secondary/60 bg-secondary/20 text-white shadow-[0_0_12px_rgba(0,217,255,0.35)]'
                        : 'border-theme text-theme-muted hover:border-theme-strong hover:text-theme'
                    }`}
                    title={
                      isDarkMode
                        ? 'Switch to Light Mode'
                        : 'Switch to Dark Mode'
                    }
                  >
                    {isDarkMode ? (
                      <SunIcon className="h-4 w-4" />
                    ) : (
                      <MoonIcon className="h-4 w-4" />
                    )}
                    Theme
                  </button>
                </div>

                {showShare && (
                  <div className="rounded-2xl border border-theme bg-theme-surface p-4">
                    <RoomSharePanel url={shareUrl} onCopy={onCopyShareLink} />
                  </div>
                )}
              </div>

              <h4 className="border-theme border-b pb-2 font-display text-[10px] text-theme-muted tracking-[0.3em]">
                Room Control
              </h4>

              {room?.mode === 'host' && (
                <div className="rounded-lg border border-secondary/30 bg-secondary/10 p-3">
                  <div className="flex items-center gap-2">
                    <div className="h-2 w-2 rounded-full bg-secondary"></div>
                    <span className="text-secondary text-sm">
                      Host Mode Active
                    </span>
                  </div>
                  <p className="mt-1 text-theme-muted text-xs">
                    In host mode, only the host can skip songs. Skip settings
                    are disabled.
                  </p>
                </div>
              )}

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`text-sm ${
                      (room?.hasPassword && !isAdmin) || room?.mode === 'host'
                        ? 'text-theme-subtle'
                        : 'text-theme'
                    }`}
                  >
                    Allow Skip
                  </span>
                  <span className="text-[10px] text-theme-muted">
                    {room?.mode === 'host'
                      ? 'Host controls skipping'
                      : 'Anyone can skip'}
                  </span>
                </div>
                <button
                  disabled={
                    (room?.hasPassword && !isAdmin) || room?.mode === 'host'
                  }
                  onClick={() =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      skipAllowed: !room.settings.skipAllowed,
                    })
                  }
                  className={`relative h-6 w-12 cursor-pointer rounded-full border border-theme transition-colors ${
                    room?.settings.skipAllowed
                      ? 'bg-secondary'
                      : 'bg-theme-surface opacity-50'
                  } ${
                    (room?.hasPassword && !isAdmin) || room?.mode === 'host'
                      ? 'cursor-not-allowed opacity-30 grayscale'
                      : ''
                  }`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.skipAllowed ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`text-sm ${
                      (room?.hasPassword && !isAdmin) || room?.mode === 'host'
                        ? 'text-theme-subtle'
                        : 'text-theme'
                    }`}
                  >
                    Democratic Skip
                  </span>
                  <span className="text-[10px] text-theme-muted">
                    {room?.mode === 'host'
                      ? 'Host decides skipping'
                      : 'Require votes'}
                  </span>
                </div>
                <button
                  disabled={
                    (room?.hasPassword && !isAdmin) || room?.mode === 'host'
                  }
                  onClick={() =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      democraticSkip: !room.settings.democraticSkip,
                    })
                  }
                  className={`relative h-6 w-12 cursor-pointer rounded-full border border-theme transition-colors ${
                    room?.settings.democraticSkip
                      ? 'bg-secondary'
                      : 'bg-theme-surface opacity-50'
                  } ${
                    (room?.hasPassword && !isAdmin) || room?.mode === 'host'
                      ? 'cursor-not-allowed opacity-30 grayscale'
                      : ''
                  }`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.democraticSkip ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`text-sm ${
                      room?.hasPassword && !isAdmin
                        ? 'text-theme-subtle'
                        : 'text-theme'
                    }`}
                  >
                    Loop Queue
                  </span>
                  <span className="text-[10px] text-theme-muted">
                    Cycled back to end
                  </span>
                </div>
                <button
                  disabled={room?.hasPassword && !isAdmin}
                  onClick={() =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      loopQueue: !room.settings.loopQueue,
                    })
                  }
                  className={`relative h-6 w-12 cursor-pointer rounded-full border border-theme transition-colors ${
                    room?.settings.loopQueue
                      ? 'bg-secondary'
                      : 'bg-theme-surface opacity-50'
                  } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.loopQueue ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`text-sm ${
                      room?.hasPassword && !isAdmin
                        ? 'text-theme-subtle'
                        : 'text-theme'
                    }`}
                  >
                    Allow Duplicates
                  </span>
                  <span className="text-[10px] text-theme-muted">
                    Same song multiple times
                  </span>
                </div>
                <button
                  disabled={room?.hasPassword && !isAdmin}
                  onClick={() =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      allowDuplicates: !room.settings.allowDuplicates,
                    })
                  }
                  className={`relative h-6 w-12 cursor-pointer rounded-full border border-theme transition-colors ${
                    room?.settings.allowDuplicates
                      ? 'bg-secondary'
                      : 'bg-theme-surface opacity-50'
                  } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.allowDuplicates ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`text-sm ${
                      room?.hasPassword && !isAdmin
                        ? 'text-theme-subtle'
                        : 'text-theme'
                    }`}
                  >
                    Remove Played
                  </span>
                  <span className="text-[10px] text-theme-muted">
                    Removed after play
                  </span>
                </div>
                <button
                  disabled={room?.hasPassword && !isAdmin}
                  onClick={() =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      removeOnPlay: !room.settings.removeOnPlay,
                    })
                  }
                  className={`relative h-6 w-12 cursor-pointer rounded-full border border-theme transition-colors ${
                    room?.settings.removeOnPlay
                      ? 'bg-secondary'
                      : 'bg-theme-surface opacity-50'
                  } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.removeOnPlay ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="border-theme border-t pt-4">
                <h5 className="mb-3 font-display text-[10px] text-theme-muted tracking-[0.3em]">
                  Room Mode
                </h5>

                <div className="space-y-2">
                  <button
                    disabled={room?.hasPassword && !isAdmin}
                    onClick={() => room && updateRoom({ mode: 'server' })}
                    className={`w-full cursor-pointer rounded-xl border p-3 text-left transition-all ${
                      room?.mode === 'server'
                        ? 'border-secondary/60 bg-secondary/10 text-theme shadow-[0_0_14px_rgba(0,217,255,0.3)]'
                        : 'border-theme bg-theme-surface text-theme-muted hover:border-theme-strong'
                    } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                  >
                    <div className="mb-1 text-sm text-theme">Server Mode</div>
                    <div className="text-[10px] text-theme-muted">
                      Auto-play music 24/7. Perfect for radio stations.
                    </div>
                  </button>

                  <button
                    disabled={room?.hasPassword && !isAdmin}
                    onClick={() => room && updateRoom({ mode: 'host' })}
                    className={`w-full cursor-pointer rounded-xl border p-3 text-left transition-all ${
                      room?.mode === 'host'
                        ? 'border-primary/60 bg-primary/10 text-theme shadow-[0_0_14px_rgba(255,46,151,0.3)]'
                        : 'border-theme bg-theme-surface text-theme-muted hover:border-theme-strong'
                    } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                  >
                    <div className="mb-1 text-sm text-theme">Host Mode</div>
                    <div className="text-[10px] text-theme-muted">
                      Host controls playback. Great for parties.
                    </div>
                  </button>
                </div>
              </div>

              {!isAdmin && (
                <div className="group mt-6 flex flex-col gap-2 border-theme border-t pt-4">
                  <span className="text-sm text-theme">Admin Access</span>
                  <div className="flex gap-2">
                    <input
                      type="password"
                      value={adminPassword}
                      onChange={(e) => onAdminPasswordChange(e.target.value)}
                      placeholder={
                        displayRoom?.hasPassword
                          ? 'Login as admin'
                          : 'Add password'
                      }
                      className="flex-1 rounded-xl border border-theme bg-theme-surface px-3 py-2 text-sm text-theme outline-none transition-all focus:border-secondary/60"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') onJoinAdmin();
                      }}
                    />
                    <button
                      onClick={onJoinAdmin}
                      disabled={isAuthenticating || !adminPassword}
                      className="cursor-pointer rounded-xl bg-primary/80 px-4 py-2 text-white text-xs transition-all hover:bg-primary disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {isAuthenticating ? '...' : 'Go'}
                    </button>
                  </div>
                </div>
              )}

              {isAdmin && (
                <div className="group mt-6 border-theme border-t pt-4 text-center">
                  <span className="text-secondary text-sm">
                    ✓ You are an Admin
                  </span>
                </div>
              )}

              <p className="pt-2 text-center text-[10px] text-theme-muted italic">
                Settings sync enabled
              </p>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
};
