import type { Room, RoomSettings, RoomUpdate } from '@vibez/models';
import {
  MoonIcon,
  ShareIcon,
  SoundCloudIcon,
  SpotifyIcon,
  SunIcon,
  Toggle,
  YouTubeIcon,
} from '@vibez/ui';
import { AnimatePresence, motion } from 'framer-motion';
import { type RefObject, useEffect, useRef, useState } from 'react';
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
  roomId?: string;
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
  roomId,
  settingsMenuRef,
}: RoomSettingsMenuProps) => {
  const [wobblePassword, setWobblePassword] = useState(false);
  const adminSectionRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (wobblePassword) {
      const timer = setTimeout(() => setWobblePassword(false), 500);
      return () => clearTimeout(timer);
    }
  }, [wobblePassword]);

  // Auto-disable "Only Admin Add" if password input is cleared (only for rooms without password)
  useEffect(() => {
    if (
      !room?.hasPassword &&
      !adminPassword &&
      room?.settings.onlyAdminAddSongs
    ) {
      updateRoomSettings({
        ...room.settings,
        onlyAdminAddSongs: false,
      });
    }
  }, [
    adminPassword,
    room?.hasPassword,
    room?.settings.onlyAdminAddSongs,
    room,
    updateRoomSettings,
  ]);

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
            className="fixed top-[var(--room-header-height)] right-0 left-0 z-10 h-[calc(100dvh-var(--room-header-height))] w-full overflow-y-auto overscroll-contain border-theme border-t bg-theme-surface p-5 text-theme shadow-2xl sm:absolute sm:top-auto sm:right-0 sm:left-auto sm:mt-3 sm:max-h-[70vh] sm:w-72 sm:overflow-y-auto sm:rounded-3xl sm:border"
          >
            <div className="space-y-4">
              <div className="space-y-3 sm:hidden">
                <div className="flex items-center gap-3">
                  <button
                    onClick={onToggleShare}
                    className={`flex flex-1 items-center justify-center gap-2 rounded-xl border p-3 font-pixel text-xs transition-all ${
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
                    className={`flex flex-1 items-center justify-center gap-2 rounded-xl border p-3 font-pixel text-xs transition-all ${
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
                    <RoomSharePanel
                      url={shareUrl}
                      roomId={roomId || ''}
                      onCopy={onCopyShareLink}
                    />
                  </div>
                )}
              </div>

              <h4 className="border-theme border-b pb-2 font-pixel text-[10px] text-theme-muted tracking-[0.3em]">
                Room Control
              </h4>

              {room?.mode === 'host' && (
                <div className="rounded-lg border border-secondary/30 bg-secondary/10 p-3">
                  <div className="flex items-center gap-2">
                    <div className="h-2 w-2 rounded-full bg-secondary"></div>
                    <span className="font-pixel text-secondary text-sm">
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
                <Toggle
                  label="Allow Skip"
                  description={
                    room?.mode === 'host'
                      ? 'Host controls skipping'
                      : 'Anyone can skip'
                  }
                  disabled={
                    (room?.hasPassword && !isAdmin) || room?.mode === 'host'
                  }
                  checked={room?.settings.skipAllowed ?? false}
                  onChange={(checked) =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      skipAllowed: checked,
                    })
                  }
                  className="!border-0 !bg-transparent !p-0 w-full"
                />
              </div>

              <div className="group flex items-center justify-between">
                <Toggle
                  label="Democratic Skip"
                  description={
                    room?.mode === 'host'
                      ? 'Host decides skipping'
                      : 'Require votes'
                  }
                  disabled={
                    (room?.hasPassword && !isAdmin) || room?.mode === 'host'
                  }
                  checked={room?.settings.democraticSkip ?? false}
                  onChange={(checked) =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      democraticSkip: checked,
                    })
                  }
                  className="!border-0 !bg-transparent !p-0 w-full"
                />
              </div>

              <div className="group flex items-center justify-between">
                <Toggle
                  label="Loop Queue"
                  description="Cycled back to end"
                  disabled={room?.hasPassword && !isAdmin}
                  checked={room?.settings.loopQueue ?? false}
                  onChange={(checked) =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      loopQueue: checked,
                    })
                  }
                  className="!border-0 !bg-transparent !p-0 w-full"
                />
              </div>

              <div className="group flex items-center justify-between">
                <Toggle
                  label="Allow Duplicates"
                  description="Same song multiple times"
                  disabled={room?.hasPassword && !isAdmin}
                  checked={room?.settings.allowDuplicates ?? false}
                  onChange={(checked) =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      allowDuplicates: checked,
                    })
                  }
                  className="!border-0 !bg-transparent !p-0 w-full"
                />
              </div>

              <div className="group flex items-center justify-between">
                <Toggle
                  label="Remove Played"
                  description="Removed after play"
                  disabled={room?.hasPassword && !isAdmin}
                  checked={room?.settings.removeOnPlay ?? false}
                  onChange={(checked) =>
                    room &&
                    updateRoomSettings({
                      ...room.settings,
                      removeOnPlay: checked,
                    })
                  }
                  className="!border-0 !bg-transparent !p-0 w-full"
                />
              </div>

              <div className="group flex items-center justify-between">
                <Toggle
                  label="Admins Only Add"
                  description="Only admins add songs"
                  disabled={room?.hasPassword && !isAdmin}
                  checked={room?.settings.onlyAdminAddSongs ?? false}
                  onChange={(checked) => {
                    if (
                      room &&
                      checked &&
                      !room.hasPassword &&
                      !adminPassword
                    ) {
                      setWobblePassword(true);
                      adminSectionRef.current?.scrollIntoView({
                        behavior: 'smooth',
                        block: 'center',
                      });
                      return;
                    }

                    room &&
                      updateRoomSettings({
                        ...room.settings,
                        onlyAdminAddSongs: checked,
                      });
                  }}
                  className="!border-0 !bg-transparent !p-0 w-full"
                />
              </div>

              <div className="border-theme border-t pt-4">
                <h5 className="mb-3 font-pixel text-[10px] text-theme-muted tracking-[0.3em]">
                  Sources
                </h5>
                <div className="grid grid-cols-3 gap-2">
                  {[
                    {
                      id: 'youtube',
                      Icon: YouTubeIcon,
                      color:
                        'bg-red-500/20 border-red-500/40 text-red-500 shadow-[0_0_10px_rgba(239,68,68,0.2)]',
                    },
                    {
                      id: 'spotify',
                      Icon: SpotifyIcon,
                      color:
                        'bg-green-500/20 border-green-500/40 text-green-500 shadow-[0_0_10px_rgba(34,197,94,0.2)]',
                    },
                    {
                      id: 'soundcloud',
                      Icon: SoundCloudIcon,
                      color:
                        'bg-orange-500/20 border-orange-500/40 text-orange-500 shadow-[0_0_10px_rgba(249,115,22,0.2)]',
                    },
                  ].map(({ id, Icon, color }) => {
                    const isEnabled =
                      room?.settings.enabledSources.includes(id) ?? true;
                    return (
                      <button
                        key={id}
                        disabled={room?.hasPassword && !isAdmin}
                        onClick={() => {
                          if (!room) return;
                          const newSources = isEnabled
                            ? room.settings.enabledSources.filter(
                                (s) => s !== id,
                              )
                            : [...room.settings.enabledSources, id];
                          updateRoomSettings({
                            ...room.settings,
                            enabledSources: newSources,
                          });
                        }}
                        className={`group relative flex cursor-pointer items-center justify-center rounded-xl border py-3 transition-all ${
                          isEnabled
                            ? color
                            : 'border-theme bg-theme-surface text-theme-muted opacity-40 hover:border-theme-strong hover:opacity-60'
                        } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed grayscale' : ''}`}
                        title={`${isEnabled ? 'Disable' : 'Enable'} ${id}`}
                      >
                        <Icon className="h-6 w-6" />
                      </button>
                    );
                  })}
                </div>
              </div>

              <div className="border-theme border-t pt-4">
                <h5 className="mb-3 font-pixel text-[10px] text-theme-muted tracking-[0.3em]">
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
                    <div className="mb-1 font-pixel text-sm text-theme">
                      Server Mode
                    </div>
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
                    <div className="mb-1 font-pixel text-sm text-theme">
                      Host Mode
                    </div>
                    <div className="text-[10px] text-theme-muted">
                      Host controls playback. Great for parties.
                    </div>
                  </button>
                </div>
              </div>

              {!isAdmin && (
                <div
                  ref={adminSectionRef}
                  className={`group mt-6 flex flex-col gap-2 border-theme border-t pt-4 text-theme transition-all duration-300 ${wobblePassword ? 'rounded-xl border-red-500' : ''}`}
                >
                  <span
                    className={`font-pixel text-sm transition-colors ${wobblePassword ? 'animate-bounce text-red-500' : ''}`}
                  >
                    {wobblePassword ? 'Password required!' : 'Admin Access'}
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
                      className={`flex-1 rounded-xl border bg-theme-surface px-3 py-2 text-sm text-theme outline-none transition-all focus:border-secondary/60 ${wobblePassword ? 'border-red-500 ring-2 ring-red-500/50' : 'border-theme'}`}
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
