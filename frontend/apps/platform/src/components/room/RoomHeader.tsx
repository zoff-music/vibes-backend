import type { Room, RoomSettings, RoomUpdate, RoomUser } from '@vibez/models';
import {
  ArrowLeftIcon,
  ChevronDownIcon,
  CopyIcon,
  MoonIcon,
  SettingsIcon,
  ShareIcon,
  SunIcon,
} from '@vibez/ui';
import { AnimatePresence, motion } from 'framer-motion';
import { type RefObject } from 'react';
import { Link } from 'react-router';
import { RoomSettingsMenu } from './RoomSettingsMenu';
import { RoomSharePanel } from './RoomSharePanel';
import { UserCount } from './UserCount';

interface RoomHeaderProps {
  headerRef: RefObject<HTMLDivElement | null>;
  displayRoom: Room | null;
  roomId?: string;
  users?: RoomUser[];
  showRoomInfo: boolean;
  onToggleRoomInfo: () => void;
  isSSR: boolean;
  showShare: boolean;
  onToggleShare: () => void;
  shareUrl: string;
  onCopyShareLink: () => void;
  isDarkMode: boolean;
  onToggleDarkMode: () => void;
  showSettings: boolean;
  onToggleSettings: () => void;
  onCloseSettings: () => void;
  settingsButtonRef: RefObject<HTMLButtonElement | null>;
  settingsMenuRef: RefObject<HTMLDivElement | null>;
  room: Room | null;
  isAdmin: boolean;
  updateRoomSettings: (settings: RoomSettings) => void;
  updateRoom: (data: RoomUpdate) => Promise<Room | null> | null;
  adminPassword: string;
  onAdminPasswordChange: (value: string) => void;
  onJoinAdmin: () => void;
  isAuthenticating: boolean;
}

export const RoomHeader = ({
  headerRef,
  displayRoom,
  roomId,
  users,
  showRoomInfo,
  onToggleRoomInfo,
  isSSR,
  showShare,
  onToggleShare,
  shareUrl,
  onCopyShareLink,
  isDarkMode,
  onToggleDarkMode,
  showSettings,
  onToggleSettings,
  onCloseSettings,
  settingsButtonRef,
  settingsMenuRef,
  room,
  isAdmin,
  updateRoomSettings,
  updateRoom,
  adminPassword,
  onAdminPasswordChange,
  onJoinAdmin,
  isAuthenticating,
}: RoomHeaderProps) => {
  return (
    <div
      ref={headerRef}
      className="panel-surface sticky top-0 z-20 border-theme border-b px-4 py-4"
    >
      <div className="relative mx-auto flex max-w-7xl items-center justify-between">
        <Link
          to="/"
          className="group inline-flex cursor-pointer items-center gap-2 text-theme-muted transition-colors hover:text-theme"
        >
          <ArrowLeftIcon className="h-5 w-5 transition-transform group-hover:-translate-x-1" />
          <span className="font-display text-[10px] tracking-[0.3em]">
            Leave
          </span>
        </Link>

        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2">
          <button
            onClick={onToggleRoomInfo}
            className="flex cursor-pointer items-center justify-center gap-2 transition-opacity hover:opacity-80"
          >
            <h1 className="truncate whitespace-nowrap font-display text-sm text-theme sm:text-base">
              {displayRoom?.name || 'Loading...'}
            </h1>
            <ChevronDownIcon
              className={`h-4 w-4 text-theme-muted transition-transform ${
                showRoomInfo ? 'rotate-180' : ''
              }`}
            />
          </button>
        </div>

        <div className="flex items-center gap-2">
          <UserCount />

          <div className="hidden sm:block">
            <button
              onClick={onToggleDarkMode}
              className={`cursor-pointer rounded-xl border p-2.5 transition-all ${
                isDarkMode
                  ? 'border-secondary/60 bg-secondary/20 text-white shadow-[0_0_18px_rgba(0,217,255,0.35)]'
                  : 'border-theme text-theme-muted hover:border-theme-strong hover:text-theme'
              }`}
              title={
                isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode'
              }
            >
              {isDarkMode ? (
                <SunIcon className="h-5 w-5" />
              ) : (
                <MoonIcon className="h-5 w-5" />
              )}
            </button>
          </div>

          <div className="relative hidden sm:block">
            <button
              onClick={onToggleShare}
              className={`cursor-pointer rounded-xl border p-2.5 transition-all ${
                showShare
                  ? 'border-theme-strong bg-theme-surface text-theme'
                  : 'border-theme text-theme-muted hover:border-theme-strong hover:text-theme'
              }`}
              title="Share Room"
            >
              <ShareIcon className="h-5 w-5" />
            </button>

            <AnimatePresence>
              {showShare && (
                <motion.div
                  initial={{ opacity: 0, y: 10, scale: 0.95 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: 10, scale: 0.95 }}
                  className="panel-strong absolute right-0 z-50 mt-3 w-72 rounded-3xl p-4 shadow-2xl"
                >
                  <RoomSharePanel url={shareUrl} onCopy={onCopyShareLink} />
                </motion.div>
              )}
            </AnimatePresence>
          </div>

          <div className="relative ml-1">
            <button
              ref={settingsButtonRef}
              onClick={onToggleSettings}
              className={`cursor-pointer rounded-xl border p-2.5 transition-all ${
                showSettings
                  ? 'border-theme-strong bg-theme-surface text-theme'
                  : 'border-theme text-theme-muted hover:border-theme-strong hover:text-theme'
              }`}
              title="Room Settings"
            >
              <SettingsIcon className="h-5 w-5" />
            </button>

            <RoomSettingsMenu
              showSettings={showSettings}
              onClose={onCloseSettings}
              showShare={showShare}
              onToggleShare={onToggleShare}
              isDarkMode={isDarkMode}
              onToggleDarkMode={onToggleDarkMode}
              room={room}
              displayRoom={displayRoom}
              isAdmin={isAdmin}
              updateRoomSettings={updateRoomSettings}
              updateRoom={updateRoom}
              adminPassword={adminPassword}
              onAdminPasswordChange={onAdminPasswordChange}
              onJoinAdmin={onJoinAdmin}
              isAuthenticating={isAuthenticating}
              shareUrl={shareUrl}
              onCopyShareLink={onCopyShareLink}
              settingsMenuRef={settingsMenuRef}
            />
          </div>
        </div>
      </div>

      {showRoomInfo && (
        <div
          className={`panel-surface mt-4 rounded-2xl p-5 ${
            !isSSR ? 'animate-slide-down' : ''
          }`}
        >
          <div className="space-y-4">
            <div>
              <p className="mb-2 font-display text-[10px] text-theme-muted tracking-[0.3em]">
                Room Code
              </p>
              <div className="flex items-center gap-2">
                <code className="rounded-xl border border-theme bg-theme-surface px-4 py-2 font-mono text-sm text-theme">
                  {roomId}
                </code>
                <button
                  onClick={() => navigator.clipboard.writeText(roomId || '')}
                  className="cursor-pointer rounded-lg border border-transparent p-2 transition-colors hover:border-theme-strong hover:bg-theme-surface"
                  title="Copy code"
                >
                  <CopyIcon className="h-5 w-5 text-theme-muted" />
                </button>
              </div>
            </div>
            {users && users.length > 0 && (
              <div>
                <p className="mb-2 font-display text-[10px] text-theme-muted tracking-[0.3em]">
                  <span className="hidden sm:inline">Listeners </span>(
                  {users.length})
                </p>
                <div className="flex flex-wrap gap-2">
                  {users.slice(0, 8).map((user: RoomUser) => (
                    <div
                      key={user.id}
                      className="rounded-full border border-theme bg-theme-surface px-3 py-1.5 text-theme text-xs"
                    >
                      {user.nickname || `User ${user.id.slice(0, 4)}`}
                    </div>
                  ))}
                  {users.length > 8 && (
                    <div className="rounded-full bg-theme-surface px-3 py-1.5 text-theme-muted text-xs">
                      +{users.length - 8} more
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
