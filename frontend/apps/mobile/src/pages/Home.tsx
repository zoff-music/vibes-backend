import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

export default function Home() {
    const [roomCode, setRoomCode] = useState('');
    const navigate = useNavigate();

    const handleCreateRoom = () => {
        navigate('/room/create');
    };

    const handleJoinRoom = () => {
        if (roomCode.trim()) {
            navigate(`/room/${roomCode.trim()}`);
        }
    };

    return (
        <div className="min-h-screen bg-background px-6 flex items-center justify-center">
            <div className="w-full max-w-md">
                <div className="text-center mb-12">
                    <h1 className="text-5xl font-bold text-primary mb-2">Vibez</h1>
                    <p className="text-base text-text-muted">Shared music queue for everyone</p>
                </div>

                <div className="space-y-6">
                    <button
                        onClick={handleCreateRoom}
                        className="w-full bg-primary text-text-inverse py-4 rounded-xl text-lg font-semibold hover:opacity-90 transition-opacity"
                    >
                        Create Room
                    </button>

                    <div className="flex items-center gap-4">
                        <div className="flex-1 h-px bg-surfaceElevated" />
                        <span className="text-sm text-text-muted">or join existing</span>
                        <div className="flex-1 h-px bg-surfaceElevated" />
                    </div>

                    <div className="flex gap-2">
                        <input
                            type="text"
                            placeholder="Enter room code"
                            value={roomCode}
                            onChange={(e) => setRoomCode(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
                            className="flex-1 bg-surface rounded-lg px-4 py-4 text-text text-base placeholder:text-zinc-500 focus:outline-none focus:ring-2 focus:ring-primary"
                        />
                        <button
                            onClick={handleJoinRoom}
                            disabled={!roomCode.trim()}
                            className="bg-surface px-6 py-4 rounded-lg text-text text-base font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:opacity-90 transition-opacity"
                        >
                            Join
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}
