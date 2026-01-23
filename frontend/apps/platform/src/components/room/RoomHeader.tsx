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
      className="sticky top-0 z-20 border-ink/10 border-b-4 bg-white/95 px-4 py-5 shadow-retro backdrop-blur-lg transition-colors duration-300 dark:border-primary/20 dark:bg-dark-paper/95"
    >
      <div className="relative mx-auto flex max-w-7xl items-center justify-between">
        <Link
          to="/"
          className="group inline-flex cursor-pointer items-center gap-2 text-ink/60 transition-colors hover:text-ink dark:text-dark-text-muted dark:hover:text-dark-text"
        >
          <ArrowLeftIcon className="h-5 w-5 transition-transform group-hover:-translate-x-1" />
          <span className="font-bold text-sm tracking-wide">Leave</span>
        </Link>

        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2">
          <button
            onClick={onToggleRoomInfo}
            className="flex cursor-pointer items-center justify-center gap-2 transition-opacity hover:opacity-70"
          >
            <h1
              className="truncate whitespace-nowrap font-black text-ink text-lg dark:text-dark-text"
              style={{ fontFamily: 'Poppins' }}
            >
              {displayRoom?.name || 'Loading...'}
            </h1>
            <ChevronDownIcon
              className={`h-4 w-4 text-ink/50 transition-transform dark:text-dark-text-muted ${
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
              className={`cursor-pointer rounded-xl border-2 p-2.5 transition-all ${
                isDarkMode
                  ? 'border-primary bg-primary text-white shadow-neon-pink'
                  : 'border-ink/10 text-ink/60 hover:border-ink/20 hover:text-ink'
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
              className={`cursor-pointer rounded-xl border-2 p-2.5 transition-all ${
                showShare
                  ? 'border-ink bg-ink text-white dark:border-primary dark:bg-primary'
                  : 'border-ink/10 text-ink/60 hover:border-ink/20 hover:text-ink dark:border-primary/20 dark:text-dark-text-muted dark:hover:text-dark-text'
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
                  className="absolute right-0 z-50 mt-3 w-72 rounded-3xl border-4 border-ink bg-white p-4 shadow-2xl dark:border-primary dark:bg-dark-surface"
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
              className={`cursor-pointer rounded-xl border-2 p-2.5 transition-all ${
                showSettings
                  ? 'border-ink bg-ink text-white dark:border-primary dark:bg-primary'
                  : 'border-ink/10 text-ink/60 hover:border-ink/20 hover:text-ink dark:border-primary/20 dark:text-dark-text-muted dark:hover:text-dark-text'
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
          className={`glass-elevated mt-4 rounded-2xl border-2 border-ink/10 p-5 dark:border-primary/20 ${
            !isSSR ? 'animate-slide-down' : ''
          }`}
        >
          <div className="space-y-4">
            <div>
              <p className="mb-2 font-bold text-ink/60 text-xs uppercase tracking-widest dark:text-dark-text-muted">
                Room Code
              </p>
              <div className="flex items-center gap-2">
                <code className="rounded-xl border-2 border-ink/20 bg-surface px-4 py-2 font-bold font-mono text-ink text-sm dark:border-primary/20 dark:bg-dark-surfaceElevated dark:text-dark-text">
                  {roomId}
                </code>
                <button
                  onClick={() => navigator.clipboard.writeText(roomId || '')}
                  className="cursor-pointer rounded-lg border-2 border-transparent p-2 transition-colors hover:border-ink/10 hover:bg-ink/5"
                  title="Copy code"
                >
                  <CopyIcon className="h-5 w-5 text-ink/60" />
                </button>
              </div>
            </div>
            {users && users.length > 0 && (
              <div>
                <p className="mb-2 font-bold text-ink/60 text-xs uppercase tracking-widest">
                  <span className="hidden sm:inline">Listeners </span>(
                  {users.length})
                </p>
                <div className="flex flex-wrap gap-2">
                  {users.slice(0, 8).map((user: RoomUser) => (
                    <div
                      key={user.id}
                      className="rounded-full border border-sakura/50 bg-sakura/30 px-3 py-1.5 font-medium text-ink/70 text-xs"
                    >
                      {user.nickname || `User ${user.id.slice(0, 4)}`}
                    </div>
                  ))}
                  {users.length > 8 && (
                    <div className="rounded-full bg-ink/10 px-3 py-1.5 font-medium text-ink/50 text-xs">
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
