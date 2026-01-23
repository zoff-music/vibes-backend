import { QRCodeSVG } from 'qrcode.react';

interface RoomSharePanelProps {
  url: string;
  onCopy: () => void;
}

export const RoomSharePanel = ({ url, onCopy }: RoomSharePanelProps) => {
  return (
    <div className="space-y-4 text-center">
      <div className="inline-block rounded-2xl bg-sakura/20 p-4 ring-2 ring-ink/5">
        <QRCodeSVG
          value={url}
          size={180}
          bgColor="#fff0f2"
          fgColor="#2d3142"
          level="H"
        />
      </div>
      <div>
        <p className="mb-1 font-black text-ink text-sm dark:text-dark-text">
          Invite Friends
        </p>
        <div className="flex items-center gap-2 rounded-xl bg-ink/5 p-2 dark:bg-dark-surfaceElevated">
          <p className="flex-1 truncate text-left font-mono text-[10px] text-ink/60 dark:text-dark-text-muted">
            {url}
          </p>
          <button
            onClick={onCopy}
            className="cursor-pointer rounded-lg bg-ink p-1 px-2 font-bold text-[10px] text-white transition-all hover:scale-105 active:scale-95 dark:bg-primary"
          >
            Copy
          </button>
        </div>
      </div>
    </div>
  );
};
