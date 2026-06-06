import { useRoom } from '@vibez/api';
import type { Room } from '@vibez/models';
import {
  ArrowLeftIcon,
  CircleHalfIcon,
  MoonIcon,
  SettingsIcon,
  ShareIcon,
  SunIcon,
} from '@vibez/ui';
import { AnimatePresence, motion } from 'framer-motion';
import React, { type RefObject } from 'react';

import { RoomSettingsMenu } from './RoomSettingsMenu';
import { RoomSharePanel } from './RoomSharePanel';
import { UserCount } from './UserCount';

interface RoomHeaderProps {
  headerRef: RefObject<HTMLDivElement | null>;
  displayRoom: Room | null;
  roomId: string;
  showShare: boolean;
  onToggleShare: () => void;
  shareUrl: string;
  onCopyShareLink: () => void;
  themeId: string;
  currentTheme: any;
  onToggleDarkMode: () => void;
  showSettings: boolean;
  onToggleSettings: () => void;
  onCloseSettings: () => void;
  settingsButtonRef: RefObject<HTMLButtonElement | null>;
  settingsMenuRef: RefObject<HTMLDivElement | null>;
  adminPassword: string;
  onAdminPasswordChange: (value: string) => void;
  onJoinAdmin: () => void;
  isAuthenticating: boolean;
  onLeave: () => void;
}

export const RoomHeader = React.memo(
  ({
    headerRef,
    displayRoom,
    roomId,
    showShare,
    onToggleShare,
    shareUrl,
    onCopyShareLink,
    themeId,
    currentTheme,
    onToggleDarkMode,
    showSettings,
    onToggleSettings,
    onCloseSettings,
    settingsButtonRef,
    settingsMenuRef,
    adminPassword,
    onAdminPasswordChange,
    onJoinAdmin,
    isAuthenticating,
    onLeave,
  }: RoomHeaderProps) => {
    const { room, isAdmin, updateRoomSettings, updateRoom } = useRoom(roomId);
    return (
      <div
        ref={headerRef}
        className="panel-surface sticky top-0 z-20 border-theme border-b px-4 py-4"
      >
        <div className="relative mx-auto flex max-w-7xl items-center justify-between">
          <button
            onClick={onLeave}
            className="group inline-flex cursor-pointer items-center gap-2 text-theme-muted transition-colors hover:text-theme"
          >
            <ArrowLeftIcon className="h-5 w-5 transition-transform group-hover:-translate-x-1" />
            <span className="text-[10px] tracking-[0.3em]">Leave</span>
          </button>

          <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2">
            <h1 className="truncate whitespace-nowrap text-theme text-xs sm:text-sm">
              {displayRoom?.name || 'Loading...'}
            </h1>
          </div>

          <div className="flex items-center gap-2">
            <UserCount />

            <div className="hidden sm:block">
              <button
                onClick={onToggleDarkMode}
                className={`cursor-pointer rounded-xl border p-2.5 transition-all ${
                  themeId !== 'light'
                    ? 'border-secondary/60 bg-secondary/20 text-white shadow-[0_0_18px_rgba(0,217,255,0.35)]'
                    : 'border-theme text-theme-muted hover:border-theme-strong hover:text-theme'
                }`}
                title={`Theme: ${currentTheme.name}`}
              >
                {themeId === 'light' && <SunIcon className="h-5 w-5" />}
                {themeId === 'dark' && <MoonIcon className="h-5 w-5" />}
                {themeId === 'auto' && <CircleHalfIcon className="h-5 w-5" />}
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
                    <RoomSharePanel
                      url={shareUrl}
                      roomId={roomId || ''}
                      onCopy={onCopyShareLink}
                    />
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
                themeId={themeId}
                currentTheme={currentTheme}
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
                roomId={roomId}
                settingsMenuRef={settingsMenuRef}
              />
            </div>
          </div>
        </div>
      </div>
    );
  },
);
