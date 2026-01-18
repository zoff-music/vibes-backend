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
            // If room was created more than 10 seconds ago, it's an existing room
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
        <div className="min-h-screen bg-background px-6 pt-12 pb-8 overflow-y-auto">
            <div className="max-w-2xl mx-auto">
                <div className="mb-8">
                    <button
                        onClick={() => navigate(-1)}
                        className="text-primary text-base mb-4 hover:opacity-80"
                    >
                        ← Back
                    </button>
                    <h1 className="text-3xl font-bold text-text">Create Room</h1>
                </div>

                <div className="space-y-6">
                    <div className="space-y-2">
                        <label className="text-sm text-text-muted font-medium">Room Name</label>
                        <input
                            type="text"
                            placeholder="Friday Night Vibes"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            className="w-full bg-surface rounded-lg px-4 py-4 text-text text-base placeholder:text-zinc-500 focus:outline-none focus:ring-2 focus:ring-primary"
                        />
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm text-text-muted font-medium">
                            Admin Password (optional)
                        </label>
                        <input
                            type="password"
                            placeholder="Leave empty for no password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full bg-surface rounded-lg px-4 py-4 text-text text-base placeholder:text-zinc-500 focus:outline-none focus:ring-2 focus:ring-primary"
                        />
                    </div>

                    <div className="mt-4 space-y-4">
                        <h2 className="text-lg font-semibold text-text">Settings</h2>

                        <div className="bg-surface rounded-lg p-4 flex items-center justify-between">
                            <div className="flex-1 mr-4">
                                <div className="text-base text-text font-medium">Allow Skip</div>
                                <div className="text-sm text-text-muted mt-0.5">
                                    Users can skip the current song
                                </div>
                            </div>
                            <label className="relative inline-flex items-center cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={settings.skipAllowed}
                                    onChange={(e) => updateSetting('skipAllowed', e.target.checked)}
                                    className="sr-only peer"
                                />
                                <div className="w-11 h-6 bg-zinc-800 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-muted rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-muted"></div>
                            </label>
                        </div>

                        <div className="bg-surface rounded-lg p-4 flex items-center justify-between">
                            <div className="flex-1 mr-4">
                                <div className="text-base text-text font-medium">Democratic Skip</div>
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
                                <div className="w-11 h-6 bg-zinc-800 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-muted rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-muted"></div>
                            </label>
                        </div>

                        <div className="bg-surface rounded-lg p-4 flex items-center justify-between">
                            <div className="flex-1 mr-4">
                                <div className="text-base text-text font-medium">Loop Queue</div>
                                <div className="text-sm text-text-muted mt-0.5">
                                    Restart from beginning when queue ends
                                </div>
                            </div>
                            <label className="relative inline-flex items-center cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={settings.loopQueue}
                                    onChange={(e) => updateSetting('loopQueue', e.target.checked)}
                                    className="sr-only peer"
                                />
                                <div className="w-11 h-6 bg-zinc-800 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-muted rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-muted"></div>
                            </label>
                        </div>
                    </div>
                </div>

                {error && (
                    <div className="mt-4 bg-error/20 border border-error rounded-lg p-3">
                        <p className="text-error text-sm">{error}</p>
                    </div>
                )}

                <button
                    onClick={handleCreate}
                    disabled={!name.trim() || isLoading}
                    className="w-full mt-8 bg-primary text-text-inverse py-4 rounded-xl text-lg font-semibold disabled:opacity-50 disabled:cursor-not-allowed hover:opacity-90 transition-opacity flex items-center justify-center"
                >
                    {isLoading ? (
                        <div className="w-5 h-5 border-2 border-text-inverse border-t-transparent rounded-full animate-spin" />
                    ) : (
                        'Create Room'
                    )}
                </button>
            </div>
        </div>
    );
}
