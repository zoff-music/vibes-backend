import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import App from './App';

// Mock the player components as they might need context
vi.mock('@vibez/ui', () => ({
  SpotifyPlayer: () => <div data-testid="spotify-player">Spotify</div>,
  SoundCloudPlayer: () => <div data-testid="soundcloud-player">SoundCloud</div>,
  VideoPlayer: () => <div data-testid="video-player">Video</div>,
}));

// Mock shared store
vi.mock('@vibez/shared', () => ({
  usePlaybackStore: () => {
    // Return mock values based on selector or default
    return null; // currentSong default
  },
  safeWrap: vi.fn(),
}));

describe('App', () => {
  it('renders without crashing', () => {
    render(<App />);
    expect(screen.getByText('Zoff')).toBeInTheDocument();
  });
});
