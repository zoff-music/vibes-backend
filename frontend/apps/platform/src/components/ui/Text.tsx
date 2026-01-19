import React from 'react';

interface Props extends React.HTMLAttributes<HTMLElement> {
    variant?: 'body' | 'heading' | 'caption' | 'mono';
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl';
    color?: 'primary' | 'secondary' | 'muted' | 'error' | 'white';
    bold?: boolean;
    as?: 'p' | 'span' | 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6';
}

export const Text: React.FC<Props> = ({
    variant = 'body',
    size = 'md',
    color = 'white',
    bold = false,
    as,
    className = '',
    children,
    ...props
}) => {
    const Component = as || (variant === 'heading' ? 'h2' : 'p');

    const sizeClasses = {
        xs: 'text-xs',
        sm: 'text-sm',
        md: 'text-base',
        lg: 'text-lg',
        xl: 'text-2xl',
        xxl: 'text-3xl',
    };

    const colorClasses = {
        white: 'text-text',
        muted: 'text-text-muted',
        primary: 'text-primary',
        secondary: 'text-secondary',
        error: 'text-error',
    };

    const variantClasses = {
        body: 'font-normal',
        heading: 'font-bold',
        caption: 'font-light opacity-80',
        mono: 'font-mono',
    };

    const classes = `${sizeClasses[size]} ${colorClasses[color]} ${variantClasses[variant]} ${bold ? 'font-bold' : ''} ${className}`;

    return (
        <Component className={classes} {...props}>
            {children}
        </Component>
    );
};
