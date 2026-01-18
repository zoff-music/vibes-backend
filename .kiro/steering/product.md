# Product Overview

**Vibez** is a collaborative music queue application with synchronized playback, designed for shared listening experiences.

## Core Features

- **Collaborative Queue**: Multiple users can add songs to a shared queue
- **Synchronized Playback**: All participants hear the same song at the same time
- **Room-based Sessions**: Users join rooms to participate in shared listening
- **YouTube Integration**: V1 focuses exclusively on YouTube as the music source
- **Real-time Updates**: Server-Sent Events (SSE) keep all clients synchronized
- **Democratic Controls**: Skip voting and admin controls for room management

## User Roles

- **Room Admin**: Creates room, has full control over playback and queue management
- **Participants**: Can add songs, vote to skip, and participate in the shared experience
- **Anonymous Users**: Can join rooms without registration using nicknames

## Key Use Cases

- Group listening sessions (friends, parties, work environments)
- Collaborative music discovery
- Remote shared music experiences
- Democratic music selection in shared spaces

## Technical Constraints

- YouTube-only for V1 (extensible architecture for future sources)
- Web-based interface (React app)
- Real-time synchronization requirements
- Minimal authentication (password-based admin, anonymous participants)