import type { Room, RoomSettings, RoomUpdate } from '@vibez/models';
import { MoonIcon, ShareIcon, SunIcon } from '@vibez/ui';
import { AnimatePresence, motion } from 'framer-motion';
import { type RefObject } from 'react';
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
            className="fixed top-[var(--room-header-height)] right-0 left-0 z-10 h-[calc(100dvh-var(--room-header-height))] w-full overflow-y-auto overscroll-contain border-ink/10 border-t-2 bg-white p-5 shadow-2xl sm:absolute sm:top-auto sm:right-0 sm:left-auto sm:mt-3 sm:h-auto sm:w-72 sm:overflow-visible sm:rounded-3xl sm:border-2 dark:border-primary/20 dark:bg-dark-surface"
          >
            <div className="space-y-4">
              <div className="space-y-3 sm:hidden">
                <div className="flex items-center gap-3">
                  <button
                    onClick={onToggleShare}
                    className={`flex flex-1 items-center justify-center gap-2 rounded-xl border-2 p-3 font-bold text-xs transition-all ${
                      showShare
                        ? 'border-ink bg-ink text-white dark:border-primary dark:bg-primary'
                        : 'border-ink/10 text-ink/70 hover:border-ink/20 hover:text-ink dark:border-primary/20 dark:text-dark-text-muted dark:hover:text-dark-text'
                    }`}
                    title="Share Room"
                  >
                    <ShareIcon className="h-4 w-4" />
                    Share
                  </button>

                  <button
                    onClick={onToggleDarkMode}
                    className={`flex flex-1 items-center justify-center gap-2 rounded-xl border-2 p-3 font-bold text-xs transition-all ${
                      isDarkMode
                        ? 'border-primary bg-primary text-white shadow-neon-pink'
                        : 'border-ink/10 text-ink/70 hover:border-ink/20 hover:text-ink'
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
                  <div className="rounded-2xl border-2 border-ink/10 bg-white p-4 dark:border-primary/20 dark:bg-dark-surfaceElevated">
                    <RoomSharePanel url={shareUrl} onCopy={onCopyShareLink} />
                  </div>
                )}
              </div>

              <h4 className="border-ink/5 border-b-2 pb-2 font-black text-ink text-sm uppercase tracking-wider dark:border-primary/20 dark:text-dark-text">
                Room Control
              </h4>

              {room?.mode === 'host' && (
                <div className="rounded-lg border border-secondary/20 bg-secondary/10 p-3 dark:border-secondary/30 dark:bg-secondary/20">
                  <div className="flex items-center gap-2">
                    <div className="h-2 w-2 rounded-full bg-secondary"></div>
                    <span className="font-bold text-secondary text-sm dark:text-secondary">
                      Host Mode Active
                    </span>
                  </div>
                  <p className="mt-1 text-ink/70 text-xs dark:text-dark-text-muted">
                    In host mode, only the host can skip songs. Skip settings
                    are disabled.
                  </p>
                </div>
              )}

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`font-bold text-sm ${(room?.hasPassword && !isAdmin) || room?.mode === 'host' ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                  >
                    Allow Skip
                  </span>
                  <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
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
                  className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.skipAllowed ? 'bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${(room?.hasPassword && !isAdmin) || room?.mode === 'host' ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.skipAllowed ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`font-bold text-sm ${(room?.hasPassword && !isAdmin) || room?.mode === 'host' ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                  >
                    Democratic Skip
                  </span>
                  <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
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
                  className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.democraticSkip ? 'bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${(room?.hasPassword && !isAdmin) || room?.mode === 'host' ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.democraticSkip ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`font-bold text-sm ${room?.hasPassword && !isAdmin ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                  >
                    Loop Queue
                  </span>
                  <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
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
                  className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.loopQueue ? 'bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.loopQueue ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`font-bold text-sm ${room?.hasPassword && !isAdmin ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                  >
                    Allow Duplicates
                  </span>
                  <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
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
                  className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.allowDuplicates ? 'bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.allowDuplicates ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="group flex items-center justify-between">
                <div className="flex flex-col">
                  <span
                    className={`font-bold text-sm ${room?.hasPassword && !isAdmin ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                  >
                    Remove Played
                  </span>
                  <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
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
                  className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.removeOnPlay ? 'bg-ink dark:bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                >
                  <div
                    className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.removeOnPlay ? 'right-1.5' : 'left-1.5'}`}
                  />
                </button>
              </div>

              <div className="border-ink/5 border-t-2 pt-4 dark:border-primary/20">
                <h5 className="mb-3 font-black text-ink text-xs uppercase tracking-wider dark:text-dark-text">
                  Room Mode
                </h5>

                <div className="space-y-2">
                  <button
                    disabled={room?.hasPassword && !isAdmin}
                    onClick={() => room && updateRoom({ mode: 'server' })}
                    className={`w-full cursor-pointer rounded-xl border-2 p-3 text-left transition-all ${
                      room?.mode === 'server'
                        ? 'border-primary bg-primary/10 text-ink dark:border-primary dark:bg-primary/20 dark:text-dark-text'
                        : 'border-ink/10 bg-surface text-ink/60 hover:border-ink/20 dark:border-primary/20 dark:bg-dark-surfaceElevated dark:text-dark-text-muted dark:hover:border-primary/30'
                    } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                  >
                    <div className="mb-1 font-bold text-sm">Server Mode</div>
                    <div className="text-[10px] opacity-70">
                      Auto-play music 24/7. Perfect for radio stations.
                    </div>
                  </button>

                  <button
                    disabled={room?.hasPassword && !isAdmin}
                    onClick={() => room && updateRoom({ mode: 'host' })}
                    className={`w-full cursor-pointer rounded-xl border-2 p-3 text-left transition-all ${
                      room?.mode === 'host'
                        ? 'border-secondary bg-secondary/10 text-ink dark:border-secondary dark:bg-secondary/20 dark:text-dark-text'
                        : 'border-ink/10 bg-surface text-ink/60 hover:border-ink/20 dark:border-primary/20 dark:bg-dark-surfaceElevated dark:text-dark-text-muted dark:hover:border-primary/30'
                    } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                  >
                    <div className="mb-1 font-bold text-sm">Host Mode</div>
                    <div className="text-[10px] opacity-70">
                      Host controls playback. Great for parties.
                    </div>
                  </button>
                </div>
              </div>

              {!isAdmin && (
                <div className="group mt-6 flex flex-col gap-2 border-ink/5 border-t-2 pt-4 dark:border-primary/20">
                  <span className="font-bold text-ink text-sm dark:text-dark-text">
                    Admin Access
                  </span>
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
                      className="flex-1 rounded-xl border-2 border-ink/10 bg-white px-3 py-2 text-ink text-sm outline-none transition-all focus:border-primary/50 dark:border-primary/20 dark:bg-dark-paper dark:text-dark-text"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') onJoinAdmin();
                      }}
                    />
                    <button
                      onClick={onJoinAdmin}
                      disabled={isAuthenticating || !adminPassword}
                      className="cursor-pointer rounded-xl bg-ink p-2 px-4 font-bold text-white text-xs transition-all hover:bg-primary disabled:cursor-not-allowed disabled:opacity-50 dark:bg-primary"
                    >
                      {isAuthenticating ? '...' : 'Go'}
                    </button>
                  </div>
                </div>
              )}

              {isAdmin && (
                <div className="group mt-6 border-ink/5 border-t-2 pt-4 text-center dark:border-primary/20">
                  <span className="font-bold text-matcha text-sm">
                    ✓ You are an Admin
                  </span>
                </div>
              )}

              <p className="pt-2 text-center font-black text-[10px] text-ink/30 italic dark:text-dark-text-subtle">
                Settings sync enabled
              </p>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
};
