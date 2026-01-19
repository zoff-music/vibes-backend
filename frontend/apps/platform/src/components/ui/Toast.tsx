import { useState, useEffect } from 'react';

interface ToastProps {
    message: string;
    type?: 'success' | 'error' | 'info';
    duration?: number;
    onClose: () => void;
}

export const Toast = ({ message, type = 'info', duration = 3000, onClose }: ToastProps) => {
    const [isVisible, setIsVisible] = useState(true);

    useEffect(() => {
        const timer = setTimeout(() => {
            setIsVisible(false);
            setTimeout(onClose, 300); // Wait for fade out animation
        }, duration);

        return () => clearTimeout(timer);
    }, [duration]); // Removed onClose to prevent timer reset during parent re-renders

    const bgColors = {
        success: 'bg-matcha border-matcha/20 text-ink',
        error: 'bg-error border-error/20 text-white',
        info: 'bg-sakura/80 border-sakura text-ink',
    };

    return (
        <div
            className={`fixed bottom-8 left-1/2 -translate-x-1/2 z-50 transition-all duration-300 transform ${isVisible ? 'translate-y-0 opacity-100' : 'translate-y-4 opacity-0'
                }`}
        >
            <div className={`px-6 py-3 rounded-2xl border-2 shadow-retro flex items-center gap-3 backdrop-blur-md ${bgColors[type]}`}>
                {type === 'success' && (
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                )}
                <span className="font-bold tracking-wide">{message}</span>
            </div>
        </div>
    );
};
