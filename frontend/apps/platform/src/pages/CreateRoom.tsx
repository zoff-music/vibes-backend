import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '@vibez/api';

const DEFAULT_SETTINGS = {
    skipAllowed: true,
    democraticSkip: true,
    loopQueue: false,
};

export default function CreateRoom() {
    const [name, setName] = useState('');
    const [mode, setMode] = useState<'server' | 'host'>('server');
    const [password, setPassword] = useState('');
    const [settings, setSettings] = useState(DEFAULT_SETTINGS);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const navigate = useNavigate();

    const handleCreate = async () => {
        if (!name.trim() || isLoading) return;

        setIsLoading(true);
        setError(null);

        const [err, room] = await api.post('/rooms', null, {
            name: name.trim(),
            password: password || undefined,
            mode,
        });

        if (err) {
            console.error('Failed to create room:', err);
            setError(err.message || 'Failed to create room');
            setIsLoading(false);
            return;
        }

        if (room) {
            const createdAt = new Date(room.createdAt);
            const now = new Date();
            const isExisting = now.getTime() - createdAt.getTime() > 10000;

            if (isExisting) {
                alert('Welcome! That room already exists, welcome!');
            }

            navigate(`/room/${room.id}`, { replace: true });
        }
    };

    const updateSetting = <K extends keyof typeof settings>(
        key: K,
        value: (typeof settings)[K]
    ) => {
        setSettings((prev) => ({ ...prev, [key]: value }));
    };

    return (
        <div className="min-h-screen px-4 pt-8 pb-12 overflow-y-auto">
            <div className="max-w-xl mx-auto animate-fade-in">
                {/* Header */}
                <div className="mb-8">
                    <button
                        onClick={() => navigate(-1)}
                        className="inline-flex items-center gap-2 text-ink/60 hover:text-ink mb-6 transition-colors group"
                    >
                        <svg className="w-5 h-5 group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
                        </svg>
                        <span className="text-sm font-bold tracking-wide">Back</span>
                    </button>
                    <h1 className="text-4xl font-black text-ink mb-2" style={{ fontFamily: 'Poppins' }}>Create Session</h1>
                    <p className="text-ink/60 mt-2 font-medium">Set up your shared music room</p>
                    <p className="text-sm jp-art text-ink/40 mt-1">セッションを作成</p>
                </div>

                {/* Form */}
                <div className="space-y-5">
                    {/* Room name */}
                    <div className="glass rounded-2xl p-6 border-2 border-ink/10">
                        <label className="text-sm font-bold text-ink/80 mb-3 block tracking-wide">
                            Session Name
                        </label>
                        <input
                            type="text"
                            placeholder="Friday Night Vibes"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
                            className="w-full bg-surface rounded-xl px-4 py-4 text-base text-ink placeholder:text-ink/40 border-2 border-ink/20 focus:outline-hidden focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] transition-all"
                            autoFocus
                        />
                    </div>

                    {/* Room Mode */}
                    <div className="glass rounded-2xl p-6 border-2 border-ink/10">
                        <label className="text-sm font-bold text-ink/80 mb-3 block tracking-wide">
                            Room Mode
                        </label>
                        <div className="grid grid-cols-2 gap-4">
                            <button
                                type="button"
                                onClick={() => setMode('server')}
                                className={`p-4 rounded-xl border-2 text-left transition-all ${
                                    mode === 'server' 
                                        ? 'bg-primary/10 border-primary text-ink' 
                                        : 'bg-surface border-ink/10 text-ink/60 hover:border-ink/20'
                                }`}
                            >
                                <div className="font-bold mb-1">Server Mode</div>
                                <div className="text-xs opacity-70">Auto-play music 24/7. Perfect for radio stations.</div>
                            </button>
                            <button
                                type="button"
                                onClick={() => setMode('host')}
                                className={`p-4 rounded-xl border-2 text-left transition-all ${
                                    mode === 'host' 
                                        ? 'bg-secondary/10 border-secondary text-ink' 
                                        : 'bg-surface border-ink/10 text-ink/60 hover:border-ink/20'
                                }`}
                            >
                                <div className="font-bold mb-1">Host Mode</div>
                                <div className="text-xs opacity-70">Host controls playback. Great for parties.</div>
                            </button>
                        </div>
                    </div>

                    {/* Optional password */}
                    <div className="glass rounded-2xl p-6 border-2 border-ink/10">
                        <label className="text-sm font-bold text-ink/80 mb-3 block tracking-wide">
                            Admin Password <span className="text-ink/40 font-normal">(optional)</span>
                        </label>
                        <input
                            type="password"
                            placeholder="For room control"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full bg-surface rounded-xl px-4 py-4 text-base text-ink placeholder:text-ink/40 border-2 border-ink/20 focus:outline-hidden focus:border-secondary focus:shadow-[0_0_0_3px_rgba(0,217,255,0.1)] transition-all"
                        />
                        <p className="text-xs text-ink/50 mt-3 font-medium">
                            Leave empty to allow anyone to control playback
                        </p>
                    </div>

                    {/* Settings */}
                    <div className="space-y-3 mt-8">
                        <h2 className="text-sm font-bold text-ink/70 uppercase tracking-widest mb-4">
                            Playback Settings
                        </h2>

                        <div className="glass rounded-2xl p-5 flex items-center justify-between group hover:shadow-retro transition-all border-2 border-ink/10">
                            <div className="flex-1 mr-4">
                                <div className="text-base font-bold text-ink">Allow Skip</div>
                                <div className="text-sm text-ink/60 mt-0.5 font-medium">
                                    Anyone can skip songs
                                </div>
                            </div>
                            <label className="relative inline-flex items-center cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={settings.skipAllowed}
                                    onChange={(e) => updateSetting('skipAllowed', e.target.checked)}
                                    className="sr-only peer"
                                />
                                <div className="w-12 h-7 bg-ink/10 peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-primary/30 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-ink/60 after:rounded-full after:h-6 after:w-6 after:transition-all peer-checked:bg-primary peer-checked:after:bg-white shadow-retro"></div>
                            </label>
                        </div>

                        <div className="glass rounded-2xl p-5 flex items-center justify-between group hover:shadow-retro transition-all border-2 border-ink/10">
                            <div className="flex-1 mr-4">
                                <div className="text-base font-bold text-ink">Democratic Skip</div>
                                <div className="text-sm text-ink/60 mt-0.5 font-medium">
                                    Require votes to skip
                                </div>
                            </div>
                            <label className="relative inline-flex items-center cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={settings.democraticSkip}
                                    onChange={(e) => updateSetting('democraticSkip', e.target.checked)}
                                    className="sr-only peer"
                                />
                                <div className="w-12 h-7 bg-ink/10 peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-primary/30 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-ink/60 after:rounded-full after:h-6 after:w-6 after:transition-all peer-checked:bg-primary peer-checked:after:bg-white shadow-retro"></div>
                            </label>
                        </div>

                        <div className="glass rounded-2xl p-5 flex items-center justify-between group hover:shadow-retro transition-all border-2 border-ink/10">
                            <div className="flex-1 mr-4">
                                <div className="text-base font-bold text-ink">Loop Queue</div>
                                <div className="text-sm text-ink/60 mt-0.5 font-medium">
                                    Restart when queue ends
                                </div>
                            </div>
                            <label className="relative inline-flex items-center cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={settings.loopQueue}
                                    onChange={(e) => updateSetting('loopQueue', e.target.checked)}
                                    className="sr-only peer"
                                />
                                <div className="w-12 h-7 bg-ink/10 peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-primary/30 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-ink/60 after:rounded-full after:h-6 after:w-6 after:transition-all peer-checked:bg-primary peer-checked:after:bg-white shadow-retro"></div>
                            </label>
                        </div>
                    </div>
                </div>

                {/* Error */}
                {error && (
                    <div className="mt-6 glass-elevated rounded-2xl p-5 border-2 border-error/30 animate-scale-in">
                        <div className="flex items-start gap-3">
                            <svg className="w-5 h-5 text-error mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            <p className="text-error text-sm flex-1 font-medium">{error}</p>
                        </div>
                    </div>
                )}

                {/* Create button */}
                <button
                    onClick={handleCreate}
                    disabled={!name.trim() || isLoading}
                    className="w-full mt-8 bg-primary hover:bg-primary-muted disabled:bg-ink/10 disabled:text-ink/30 text-white py-4 rounded-2xl text-base font-black transition-all disabled:cursor-not-allowed hover:shadow-retro-pink active:scale-[0.98] flex items-center justify-center gap-2 tracking-wide"
                    style={{ fontFamily: 'Poppins' }}
                >
                    {isLoading ? (
                        <>
                            <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                            <span>Creating...</span>
                        </>
                    ) : (
                        <>
                            <span>Create Session</span>
                            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                <path strokeLinecap="round" strokeLinejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
                            </svg>
                        </>
                    )}
                </button>
            </div>
        </div>
    );
}
