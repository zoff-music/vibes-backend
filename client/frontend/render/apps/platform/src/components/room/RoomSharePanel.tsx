import { QRCodeSVG } from 'qrcode.react';

interface RoomSharePanelProps {
  url: string;
  roomId: string;
  onCopy: () => void;
}

export const RoomSharePanel = ({
  url,
  roomId,
  onCopy,
}: RoomSharePanelProps) => {
  const handleCopyRoomId = () => {
    navigator.clipboard.writeText(roomId);
    // We could trigger a toast here if we had access to the toast system,
    // but for now we follow the existing pattern.
  };

  return (
    <div className="space-y-6 text-center">
      <div className="inline-block rounded-2xl border border-theme bg-theme-surface p-4">
        <QRCodeSVG
          value={url}
          size={180}
          bgColor="#ffffff"
          fgColor="#2a1840"
          level="H"
        />
      </div>

      <div className="space-y-4">
        <div>
          <p className="mb-2 font-display text-[10px] text-theme-muted tracking-[0.2em]">
            Room Code
          </p>
          <div className="flex items-center gap-2 rounded-xl border border-theme bg-theme-surface p-2">
            <code className="flex-1 text-left font-mono text-theme text-xs">
              {roomId}
            </code>
            <button
              onClick={handleCopyRoomId}
              className="cursor-pointer rounded-lg bg-theme-surface px-3 py-1.5 text-[10px] text-theme transition-all hover:bg-theme active:scale-95"
            >
              Copy
            </button>
          </div>
        </div>

        <div>
          <p className="mb-2 font-display text-[10px] text-theme-muted tracking-[0.2em]">
            Invite Link
          </p>
          <div className="flex items-center gap-2 rounded-xl border border-theme bg-theme-surface p-2">
            <p className="flex-1 truncate text-left font-mono text-[10px] text-theme-muted">
              {url}
            </p>
            <button
              onClick={onCopy}
              className="cursor-pointer rounded-lg bg-theme-surface px-3 py-1.5 text-[10px] text-theme transition-all hover:bg-theme active:scale-95"
            >
              Copy
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
