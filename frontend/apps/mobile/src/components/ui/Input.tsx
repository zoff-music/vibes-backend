import React from 'react';

interface Props extends React.InputHTMLAttributes<HTMLInputElement> {
    label?: string;
    error?: string;
}

export const Input: React.FC<Props> = ({ label, error, className = '', ...props }) => {
    return (
        <div className="w-full mb-4">
            {label && (
                <label className="block text-sm text-text-muted font-medium mb-2 ml-1">
                    {label}
                </label>
            )}
            <input
                className={`w-full bg-surface rounded-lg px-4 py-3 text-text text-base placeholder:text-zinc-500 focus:outline-hidden focus:ring-2 focus:ring-primary border ${
                    error ? 'border-error' : 'border-surfaceElevated'
                } ${className}`}
                {...props}
            />
            {error && (
                <p className="text-xs text-error mt-1 ml-1">{error}</p>
            )}
        </div>
    );
};
