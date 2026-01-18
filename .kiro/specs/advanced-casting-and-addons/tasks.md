# Implementation Plan: Advanced Casting System

## Overview

This implementation plan focuses on adding advanced casting capabilities (Google Cast and AirPlay) to the existing Vibez collaborative music queue application. The implementation uses existing data structures and backend infrastructure while adding casting-specific functionality.

The implementation follows a layered approach: foundational casting interfaces first, then Google Cast integration, followed by AirPlay support, synchronization systems, and finally integration with existing Vibez infrastructure.

## Tasks

- [ ] 1. Set up foundational casting interfaces using existing data structures
  - [x] 1.1 Extend existing Room model in backend/vibe/vibe.go for casting
    - Add casting-related fields to existing Room struct
    - Add CastSession and CastDevice types to vibe.go
    - Use existing database patterns without new tables
    - _Requirements: 1.1, 1.2_

  - [~] 1.2 Define TypeScript casting interfaces in frontend
    - Create CastManager, CastDevice, and CastSession interfaces
    - Add casting types to existing API schemas
    - Extend existing Zustand stores for casting state
    - Ensure dark mode compatibility for all casting UI components
    - _Requirements: 1.1, 1.2_

- [ ] 2. Implement Google Cast integration
  - [~] 2.1 Create Google Cast SDK integration layer
    - Load Google Cast Web Sender SDK with proper configuration
    - Implement device discovery and connection management
    - Create cast session lifecycle management
    - _Requirements: 1.1, 1.2_

  - [~] 2.2 Build custom Cast Receiver application
    - Create HTML5 Cast Receiver app with queue display
    - Implement media session handling and custom messaging
    - Display current song, queue, and room information
    - _Requirements: 1.3, 3.1, 3.2_

  - [~] 2.3 Implement Cast UI components for sender application
    - Create cast button and device selection UI with dark mode support
    - Build cast status display and controls using Tailwind CSS v4
    - Implement queue display for cast devices with responsive design
    - _Requirements: 1.3, 3.1, 3.2_

- [ ] 3. Implement AirPlay integration
  - [~] 3.1 Create AirPlay manager using WebKit APIs
    - Implement AirPlay device discovery using Remote Playback API
    - Create connection management for AirPlay devices
    - Handle Safari-specific WebKit AirPlay APIs
    - _Requirements: 2.1, 2.2_

  - [~] 3.2 Add cross-browser AirPlay compatibility layer
    - Implement fallback for non-Safari browsers
    - Create unified AirPlay interface across browsers
    - Handle browser-specific limitations gracefully
    - _Requirements: 2.1, 2.2, 9.3_

- [~] 4. Checkpoint - Ensure basic casting functionality works
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Implement synchronization and timing systems
  - [~] 5.1 Create high-precision synchronization manager
    - Implement sub-100ms synchronization for casting
    - Create drift detection and correction algorithms
    - Add network latency compensation
    - _Requirements: 8.1, 8.3_

  - [~] 5.2 Extend existing SSE system for casting events
    - Add casting events to existing SSE broadcasting
    - Implement cast state synchronization across participants
    - Use existing internalpubsub client patterns
    - _Requirements: 8.2, 8.5_

- [ ] 6. Implement error handling and graceful degradation
  - [~] 6.1 Add casting error handling and recovery
    - Implement exponential backoff for reconnection attempts
    - Create graceful fallback to web-only playback
    - Add clear user feedback for connection issues
    - _Requirements: 1.5, 2.5_

  - [~] 6.2 Implement legacy client compatibility
    - Add graceful degradation for older clients
    - Create feature detection and progressive enhancement
    - Ensure core functionality works without casting features
    - _Requirements: 9.3_

- [ ] 7. Implement casting configuration using existing patterns
  - [~] 7.1 Add casting preferences to existing room creation
    - Extend existing room creation API endpoints
    - Add casting device preferences to room settings
    - Use existing room management patterns
    - _Requirements: 10.1, 10.3_

  - [~] 7.2 Create casting controls for room admins
    - Add casting controls to existing admin UI
    - Implement real-time casting enable/disable
    - Use existing admin permission patterns
    - _Requirements: 10.2_

- [ ] 8. Ensure backward compatibility and performance
  - [~] 8.1 Validate existing functionality preservation
    - Test that Host Mode and Follower Mode work unchanged
    - Verify API endpoint backward compatibility
    - Ensure system operates identically when casting is disabled
    - _Requirements: 9.1, 9.2, 9.4_

  - [~] 8.2 Performance optimization and monitoring
    - Optimize casting synchronization performance
    - Ensure no performance degradation for non-casting sessions
    - Use existing monitoring patterns
    - _Requirements: 9.5_

- [ ] 9. Integration and final wiring
  - [~] 9.1 Wire casting system with existing room management
    - Integrate cast manager with existing room lifecycle
    - Connect casting controls to existing playback engine
    - Use existing SSE patterns for cast state updates
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [~] 9.2 Complete cast receiver application integration
    - Finalize custom cast receiver with queue display
    - Test end-to-end casting workflows
    - Verify cast receiver handles all existing room events
    - _Requirements: 1.3, 3.1, 3.2_

- [~] 10. Final checkpoint - Ensure all casting functionality works
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at major milestones
- Implementation uses existing Vibez data structures and patterns
- Focus on casting functionality only - addon system deferred
- Maintains strict backward compatibility with existing functionality
- Uses existing backend patterns (vibe.go types, client interfaces, handler patterns)
- Uses existing frontend patterns (Zustand stores, wiretyped API, Tailwind CSS v4)
- Implements dark mode support for all casting UI components
- Follows mobile-first responsive design principles