interface PlayIconProps {
  className?: string;
}

export const PlayIcon = ({ className }: PlayIconProps) => (
  <svg
    className={className}
    fill="currentColor"
    viewBox="0 0 24 24"
    aria-hidden="true"
  >
    <path d="M8 5v14l11-7z" />
  </svg>
);
