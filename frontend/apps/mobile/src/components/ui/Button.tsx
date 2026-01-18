import React from 'react';

interface Props {
    title: string;
    onClick: () => void;
    variant?: 'primary' | 'secondary' | 'outline-solid' | 'ghost';
    size?: 'sm' | 'md' | 'lg';
    loading?: boolean;
    disabled?: boolean;
    className?: string;
}

export const Button: React.FC<Props> = ({
    title,
    onClick,
    variant = 'primary',
    size = 'md',
    loading = false,
    disabled = false,
    className = '',
}) => {
    const baseClasses = 'inline-flex items-center justify-center rounded-lg font-medium transition-opacity disabled:opacity-50 disabled:cursor-not-allowed hover:opacity-90';
    
    const variantClasses = {
        primary: 'bg-primary text-text-inverse',
        secondary: 'bg-secondary text-text-inverse',
        outline: 'bg-transparent border border-primary text-primary',
        ghost: 'bg-transparent text-primary',
    };

    const sizeClasses = {
        sm: 'px-3 py-1.5 text-sm',
        md: 'px-5 py-2.5 text-base',
        lg: 'px-7 py-3.5 text-lg',
    };

    const classes = `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`;

    return (
        <button
            onClick={onClick}
            disabled={disabled || loading}
            className={classes}
        >
            {loading ? (
                <div className="w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin" />
            ) : (
                title
            )}
        </button>
    );
};
