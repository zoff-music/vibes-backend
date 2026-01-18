import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';

const DEFAULT_SETTINGS = {
    skipAllowed: true,
    democraticSkip: true,
    loopQueue: false,
};

export default function CreateRoom() {
    const [name, setName] = useState('');
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
                        className="inline-flex items-center gap-2 text-text-muted hover:text-white mb-6 transition-colors group"
                    >
                        <svg className="w-5 h-5 group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                        </svg>
                        <span className="text-sm font-medium">Back</span>
                    </button>
                    <h1 className="text-3xl font-bold">Create Session</h1>
                    <p className="text-text-muted mt-2">Set up your shared music room</p>
                </div>

                {/* Form */}
                <div className="space-y-6">
                    {/* Room name */}
                    <div className="glass rounded-xl p-6">
                        <label className="text-sm font-medium text-text-muted mb-3 block">
                            Session Name
                        </label>
                        <input
                            type="text"
                            placeholder="Friday Night Vibes"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
                            className="w-full bg-surfaceElevated/50 backdrop-blur-sm rounded-lg px-4 py-3.5 text-base text-white placeholder:text-text-subtle border border-border focus:outline-none focus:border-primary/50 focus:ring-2 focus:ring-primary/20 transition-all"
                            autoFocus
                        />
                    </div>

                    {/* Optional password */}
                    <div className="glass rounded-xl p-6">
                        <label className="text-sm font-medium text-text-muted mb-3 block">
                            Admin Password <span className="text-text-subtle">(optional)</span>
                        </label>
                        <input
                            type="password"
                            placeholder="For room control"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full bg-surfaceElevated/50 backdrop-blur-sm rounded-lg px-4 py-3.5 text-base text-white placeholder:text-text-subtle border border-border focus:outline-none focus:border-primary/50 focus:ring-2 focus:ring-primary/20 transition-all"
                        />
                        <p className="text-xs text-text-subtle mt-2">
                            Leave empty to allow anyone to control playback
                        </p>
                    </div>

                    {/* Settings */}
                    <div className="space-y-3">
                        <h2 className="text-sm font-medium text-text-muted uppercase tracking-wider">
                            Playback Settings
                        </h2>

                        <div className="glass rounded-xl p-5 flex items-center justify-between group hover:bg-surfaceHover transition-colors">
                            <div className="flex-1 mr-4">
                                <div className="text-base font-medium">Allow Skip</div>
                                <div className="text-sm text-text-muted mt-0.5">
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
                                <div className="w-11 h-6 bg-surfaceElevated peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary/30 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white/80 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary peer-checked:after:bg-white"></div>
                            </label>
                        </div>

                        <div className="glass rounded-xl p-5 flex items-center justify-between group hover:bg-surfaceHover transition-colors">
                            <div className="flex-1 mr-4">
                                <div className="text-base font-medium">Democratic Skip</div>
                                <div className="text-sm text-text-muted mt-0.5">
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
                                <div className="w-11 h-6 bg-surfaceElevated peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary/30 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white/80 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary peer-checked:after:bg-white"></div>
                            </label>
                        </div>

                        <div className="glass rounded-xl p-5 flex items-center justify-between group hover:bg-surfaceHover transition-colors">
                            <div className="flex-1 mr-4">
                                <div className="text-base font-medium">Loop Queue</div>
                                <div className="text-sm text-text-muted mt-0.5">
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
                                <div className="w-11 h-6 bg-surfaceElevated peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary/30 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white/80 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary peer-checked:after:bg-white"></div>
                            </label>
                        </div>
                    </div>
                </div>

                {/* Error */}
                {error && (
                    <div className="mt-6 glass-elevated rounded-xl p-4 border-error/50 animate-scale-in">
                        <div className="flex items-start gap-3">
                            <svg className="w-5 h-5 text-error mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            <p className="text-error text-sm flex-1">{error}</p>
                        </div>
                    </div>
                )}

                {/* Create button */}
                <button
                    onClick={handleCreate}
                    disabled={!name.trim() || isLoading}
                    className="w-full mt-8 bg-primary hover:bg-primary-muted disabled:bg-surface disabled:text-text-subtle text-white py-4 rounded-xl text-base font-semibold transition-all disabled:cursor-not-allowed hover:shadow-glow active:scale-[0.98] flex items-center justify-center gap-2"
                >
                    {isLoading ? (
                        <>
                            <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                            <span>Creating...</span>
                        </>
                    ) : (
                        <>
                            <span>Create Session</span>
                            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                            </svg>
                        </>
                    )}
                </button>
            </div>
        </div>
    );
}
