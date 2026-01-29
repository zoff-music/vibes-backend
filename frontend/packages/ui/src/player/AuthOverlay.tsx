import { motion } from 'framer-motion';

interface AuthOverlayProps {
  provider: string;
  errorMessage?: string | null;
  onAuthorize: () => void;
}

export function AuthOverlay({
  provider,
  errorMessage,
  onAuthorize,
}: AuthOverlayProps) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="absolute inset-0 z-[999] flex items-center justify-center rounded-2xl bg-black/95 backdrop-blur-sm"
    >
      <div className="mx-4 max-w-sm rounded-[32px] border-[5px] border-primary bg-paper p-10 text-center shadow-[0_0_80px_rgba(0,0,0,0.4),0_0_40px_rgba(255,46,151,0.3)] dark:bg-dark-background">
        <h3 className="mb-2 font-black text-text text-xl tracking-tight dark:text-white">
          {errorMessage ? 'ACCESS RESTRICTED' : 'AUTHENTICATION REQUIRED'}
        </h3>
        <p className="mb-8 text-sm text-text-muted leading-relaxed dark:text-white/70">
          {errorMessage ? (
            errorMessage
          ) : (
            <>
              Connect to{' '}
              <span className="font-bold text-primary capitalize">
                {provider}
              </span>{' '}
              to enable playback in this room.
            </>
          )}
        </p>
        <button
          onClick={onAuthorize}
          className="w-full rounded-2xl bg-primary px-8 py-4 font-black text-lg text-white shadow-[0_4px_0_0_#9d124b] transition-all hover:-translate-y-1 hover:brightness-110 active:translate-y-0 active:shadow-none"
        >
          {errorMessage
            ? 'RETRY CONNECTION'
            : `CONNECT ${provider.toUpperCase()}`}
        </button>
      </div>
    </motion.div>
  );
}
