import React from 'react';

interface ToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: React.ReactNode;
  description?: React.ReactNode;
  disabled?: boolean;
  className?: string;
  id?: string;
  name?: string;
}

export const Toggle: React.FC<ToggleProps> = ({
  checked,
  onChange,
  label,
  description,
  disabled = false,
  className = '',
  id,
  name,
}) => {
  const generatedId = React.useId();
  const inputId = id || generatedId;

  const content = (
    <>
      <div className="mr-4 flex-1">
        {label && (
          <div className="font-pixel text-theme text-xs tracking-[0.2em]">
            {label}
          </div>
        )}
        {description && (
          <div className="mt-1 text-theme-muted text-xs">{description}</div>
        )}
      </div>
      <div className="relative inline-flex items-center">
        <input
          id={inputId}
          name={name}
          type="checkbox"
          checked={checked}
          onChange={(e) => !disabled && onChange(e.target.checked)}
          disabled={disabled}
          className="peer sr-only"
        />
        <div
          className={`peer h-7 w-12 rounded-full bg-black/10 shadow-[0_0_10px_rgba(0,0,0,0.1)] ring-1 ring-theme-neutral/30 transition-all after:absolute after:top-[2px] after:left-[2px] after:h-6 after:w-6 after:rounded-full after:bg-theme-subtle after:transition-all after:content-[''] peer-checked:bg-secondary peer-checked:ring-secondary peer-checked:after:translate-x-[85%] peer-checked:after:bg-white peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-secondary/30 dark:bg-white/10 dark:peer-checked:bg-secondary ${disabled ? 'cursor-not-allowed opacity-50 grayscale' : ''}
          `}
        />
      </div>
    </>
  );

  // If we have a label or description, wrap in the card style used in the app
  // Otherwise just return the toggle itself if it's used standalone
  if (label || description) {
    return (
      <label
        htmlFor={inputId}
        className={`group flex items-center justify-between rounded-2xl border border-theme bg-theme-surface p-5 transition-all ${
          !disabled
            ? 'cursor-pointer hover:border-theme-strong'
            : 'cursor-not-allowed opacity-60'
        } 
        ${className}`}
      >
        {content}
      </label>
    );
  }

  return (
    <label
      htmlFor={inputId}
      className={`relative inline-flex cursor-pointer items-center ${
        disabled ? 'cursor-not-allowed opacity-50' : ''
      } ${className}`}
    >
      <input
        id={inputId}
        name={name}
        type="checkbox"
        checked={checked}
        onChange={(e) => !disabled && onChange(e.target.checked)}
        disabled={disabled}
        className="peer sr-only"
      />
      <div
        className={`peer h-7 w-12 rounded-full bg-black/10 shadow-[0_0_10px_rgba(0,0,0,0.1)] ring-1 ring-theme-neutral/30 transition-all after:absolute after:top-[2px] after:left-[2px] after:h-6 after:w-6 after:rounded-full :after:bg-white after:bg-theme-subtle after:transition-all after:content-[''] peer-checked:bg-secondary peer-checked:ring-secondary peer-checked:after:translate-x-full peer-checked:after:bg-white peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-secondary/30 dark:bg-white/10 dark:peer-checked:bg-secondary ${disabled ? 'cursor-not-allowed opacity-50 grayscale' : ''}
        `}
      />
    </label>
  );
};
