import React from 'react';

interface Props extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export const Input: React.FC<Props> = ({
  label,
  error,
  className = '',
  ...props
}) => {
  return (
    <div className="mb-4 w-full">
      {label && (
        <label className="mb-2 ml-1 block font-medium text-sm text-text-muted">
          {label}
        </label>
      )}
      <input
        className={`w-full rounded-lg border bg-surface px-4 py-3 text-base text-text placeholder:text-zinc-500 focus:outline-hidden focus:ring-2 focus:ring-primary ${
          error ? 'border-error' : 'border-surfaceElevated'
        } ${className}`}
        {...props}
      />
      {error && <p className="mt-1 ml-1 text-error text-xs">{error}</p>}
    </div>
  );
};
