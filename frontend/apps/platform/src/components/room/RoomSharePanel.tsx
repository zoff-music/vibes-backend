import { QRCodeSVG } from 'qrcode.react';

interface RoomSharePanelProps {
  url: string;
  onCopy: () => void;
}

export const RoomSharePanel = ({ url, onCopy }: RoomSharePanelProps) => {
  return (
    <div className="space-y-4 text-center">
      <div className="inline-block rounded-2xl border border-theme bg-theme-surface p-4">
        <QRCodeSVG
          value={url}
          size={180}
          bgColor="#ffffff"
          fgColor="#2a1840"
          level="H"
        />
      </div>
      <div>
        <p className="mb-1 font-display text-theme text-xs">Invite Friends</p>
        <div className="flex items-center gap-2 rounded-xl border border-theme bg-theme-surface p-2">
          <p className="flex-1 truncate text-left font-mono text-[10px] text-theme-muted">
            {url}
          </p>
          <button
            onClick={onCopy}
            className="cursor-pointer rounded-lg bg-theme-surface px-2 py-1 text-[10px] text-theme transition-all hover:bg-theme active:scale-95"
          >
            Copy
          </button>
        </div>
      </div>
    </div>
  );
};
