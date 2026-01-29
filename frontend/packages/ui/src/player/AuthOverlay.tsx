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
      className="absolute inset-0 z-50 flex items-center justify-center rounded-2xl bg-black/80"
    >
      <div className="mx-4 max-w-sm rounded-2xl border-4 border-primary bg-dark-surface p-6 text-center shadow-2xl">
        <h3 className="mb-2 font-black text-dark-text text-lg">
          {errorMessage ? 'Access Restricted' : 'Authentication Required'}
        </h3>
        <p className="mb-6 text-dark-text-muted text-sm">
          {errorMessage ? (
            errorMessage
          ) : (
            <>
              You need to connect to{' '}
              <span className="font-bold capitalize">{provider}</span> to play
              this content.
            </>
          )}
        </p>
        <button
          onClick={onAuthorize}
          className="w-full rounded-xl bg-primary px-6 py-3 font-bold text-white shadow-retro transition-transform hover:scale-105 active:scale-95"
        >
          {errorMessage
            ? 'Fix Connection'
            : `Connect ${provider.charAt(0).toUpperCase() + provider.slice(1)}`}
        </button>
      </div>
    </motion.div>
  );
}
