import { useRoomStore } from '@vibez/shared';

export const UserCount = () => {
  const usersCount = useRoomStore((state) => state.usersCount);

  if (usersCount === 0) return null;

  return (
    <div className="flex h-11 items-center gap-1.5 rounded-xl border border-theme bg-theme-surface px-3 text-theme">
      <div className="h-2 w-2 animate-pulse rounded-full bg-secondary" />
      <span className="text-[10px] tracking-[0.2em]">
        <span className="sm:hidden">{usersCount}</span>
        <span className="hidden sm:inline">
          {usersCount} {usersCount === 1 ? 'Listener' : 'Listeners'}
        </span>
      </span>
    </div>
  );
};
