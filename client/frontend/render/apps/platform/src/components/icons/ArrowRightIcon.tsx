interface ArrowRightIconProps {
  className?: string;
}

export const ArrowRightIcon = ({ className }: ArrowRightIconProps) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    strokeWidth={2.5}
    aria-hidden="true"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      d="M13 7l5 5m0 0l-5 5m5-5H6"
    />
  </svg>
);
