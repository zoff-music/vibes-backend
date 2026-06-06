import { useEffect, useState } from 'react';
import { CheckIcon } from '../icons';

interface ToastProps {
  message: string;
  type?: 'success' | 'error' | 'info';
  duration?: number;
  onClose: () => void;
}

export const Toast = ({
  message,
  type = 'info',
  duration = 3000,
  onClose,
}: ToastProps) => {
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsVisible(false);
      setTimeout(onClose, 300);
    }, duration);

    return () => clearTimeout(timer);
  }, [duration]);

  const bgColors = {
    success: 'bg-matcha border-matcha/20 text-ink',
    error: 'bg-error border-error/20 text-white',
    info: 'bg-sakura/80 border-sakura text-ink',
  };

  return (
    <div
      className={`fixed bottom-8 left-1/2 z-50 -translate-x-1/2 transform transition-all duration-300 ${
        isVisible ? 'translate-y-0 opacity-100' : 'translate-y-4 opacity-0'
      }`}
    >
      <div
        className={`flex items-center gap-3 rounded-2xl border-2 px-6 py-3 shadow-retro backdrop-blur-md ${bgColors[type]}`}
      >
        {type === 'success' && <CheckIcon className="h-5 w-5" />}
        <span className="font-bold tracking-wide">{message}</span>
      </div>
    </div>
  );
};
