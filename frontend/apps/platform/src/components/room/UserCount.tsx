import { useRoomStore } from '../../stores/roomStore';

export const UserCount = () => {
  const usersCount = useRoomStore((state) => state.usersCount);

  if (usersCount === 0) return null;

  return (
    <div className="glass flex items-center gap-1.5 rounded-full border-2 border-ink/10 px-3 py-1.5 dark:border-primary/20">
      <div className="h-2 w-2 animate-pulse rounded-full bg-success" />
      <span className="font-medium text-xs dark:text-dark-text">
        {usersCount} {usersCount === 1 ? 'Listener' : 'Listeners'}
      </span>
    </div>
  );
};
