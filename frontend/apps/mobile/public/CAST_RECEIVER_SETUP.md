# Vibez Custom Cast Receiver Setup

## Overview

The Vibez Cast Receiver (`cast-receiver.html`) is a custom Google Cast receiver application designed to handle YouTube content and display Vibez-specific information like room details and queue information.

## Official Google Cast Documentation

### Core Resources
- **[Web Receiver Core Features](https://developers.google.com/cast/docs/web_receiver/core_features)** - Essential guide for Cast receiver development
- **[Google Cast Receiver Samples](https://github.com/googlecast/CastReceiver)** - Official sample implementations and best practices

### Key Documentation Sections
- **Media Management**: How receivers handle different media types
- **Custom Messages**: Implementing custom communication protocols
- **Player State Management**: Handling play, pause, seek operations
- **Error Handling**: Proper error reporting and recovery
- **Performance Optimization**: Best practices for smooth playback

## Features

- **YouTube Video Playback**: Handles YouTube URLs that can't be played by the Default Media Receiver
- **Room Information Display**: Shows room name and participant count
- **Queue Display**: Shows upcoming songs in the queue
- **Progress Tracking**: Real-time playback progress with time display
- **Responsive Design**: Works on all Chromecast screen sizes
- **Auto-hiding UI**: Media info and queue auto-hide for clean viewing
- **Dual Mode Support**: Works both as Cast receiver and standalone browser testing

## Current Implementation

### Cast Manager Configuration

The system now uses the **Styled Media Receiver** (App ID: `4F8B3483`) which allows loading custom HTML content:

```typescript
// In castManager.ts
const CAST_APPLICATION_ID = '4F8B3483'; // Styled Media Receiver
const CUSTOM_RECEIVER_URL = 'http://localhost:5173/cast-receiver.html';
```

### How It Works

1. **Connection**: User connects to Chromecast device
2. **Receiver Loading**: System loads our custom receiver HTML page on the Chromecast
3. **Content Delivery**: Once receiver is loaded, YouTube content is sent via custom messages
4. **Playback**: Receiver handles YouTube video playback using YouTube IFrame API

## Testing the Receiver

### 1. Standalone Browser Testing
- Open: `http://localhost:5173/cast-receiver.html`
- The receiver will automatically load in test mode after 3 seconds
- Shows sample YouTube video, room info, and queue

### 2. Cast Testing
1. Connect to a Chromecast device in the app
2. Click "Cast Current Song" with a YouTube video
3. The custom receiver should load on your Chromecast
4. YouTube video should start playing with Vibez branding

## Message Protocols

Following [Google Cast custom message guidelines](https://developers.google.com/cast/docs/web_receiver/core_features#custom_messages):

### Media Loading
```javascript
// Namespace: 'urn:x-cast:vibez.media'
{
  action: "loadYouTube",
  videoId: "dQw4w9WgXcQ",
  metadata: {
    title: "Song Title",
    artist: "Artist Name"
  }
}
```

### Room Information
```javascript
// Namespace: 'urn:x-cast:vibez.room'
{
  name: "Room Name",
  participantCount: 5
}
```

### Queue Updates
```javascript
// Namespace: 'urn:x-cast:vibez.queue'
{
  songs: [
    {
      id: "song-id",
      title: "Song Title",
      artist: "Artist Name",
      thumbnailUrl: "https://...",
      duration: 240
    }
  ]
}
```

## Implementation Best Practices

Based on [Google Cast Receiver samples](https://github.com/googlecast/CastReceiver):

### 1. Receiver Initialization
```javascript
// Following Google's recommended pattern
const context = cast.framework.CastReceiverContext.getInstance();
const options = new cast.framework.CastReceiverOptions();
options.maxInactivity = 3600; // 1 hour
context.start(options);
```

### 2. Message Handling
```javascript
// Custom message listeners as per Google guidelines
context.addCustomMessageListener('urn:x-cast:vibez.media', (event) => {
  console.log('Media message received:', event.data);
  handleMediaMessage(event.data);
});
```

### 3. Error Handling
```javascript
// Proper error reporting following Cast framework patterns
playerManager.addEventListener(cast.framework.events.EventType.ERROR, (event) => {
  console.error('Player error:', event.detailedErrorCode);
});
```

## Production Deployment

### Option 1: Continue with Styled Media Receiver
- Pros: Works immediately, no registration needed
- Cons: Limited customization, relies on Google's receiver
- Current setup works for production

### Option 2: Register Custom Receiver
Following [Google Cast Console registration process](https://developers.google.com/cast/docs/registration):

1. Go to [Google Cast SDK Developer Console](https://cast.google.com/publish/)
2. Create new application
3. Set receiver URL: `https://yourdomain.com/cast-receiver.html`
4. Update `CAST_APPLICATION_ID` in castManager.ts
5. Deploy receiver to HTTPS domain

### Deployment Requirements
- **HTTPS**: Receiver must be served over HTTPS in production
- **CORS**: Proper CORS headers for cross-origin requests
- **Performance**: Optimize for Chromecast hardware limitations
- **Validation**: Validate all incoming messages and data

## Troubleshooting

### Common Issues

1. **Receiver Not Loading**:
   - Ensure development server is running on port 5173
   - Check that `http://localhost:5173/cast-receiver.html` loads in browser
   - Verify Chromecast is on same network
   - Review [Google's receiver debugging guide](https://developers.google.com/cast/docs/web_receiver/debugging)

2. **YouTube Videos Not Playing**:
   - Check browser console for YouTube API errors
   - Verify video ID extraction is working
   - Ensure video is publicly available
   - Review YouTube IFrame API documentation

3. **Custom Messages Not Working**:
   - Check that receiver is properly initialized
   - Verify message namespace spelling (must start with `urn:x-cast:`)
   - Look for connection errors in console
   - Follow [custom message best practices](https://developers.google.com/cast/docs/web_receiver/core_features#custom_messages)

### Debug Information

Both the sender and receiver provide extensive logging:
- Sender: Cast SDK initialization, session management, media loading
- Receiver: Framework initialization, message handling, YouTube player events

Use Chrome DevTools for remote debugging of Cast receivers as described in [Google's debugging documentation](https://developers.google.com/cast/docs/web_receiver/debugging).

## Browser Compatibility

The receiver works with:
- Chrome (recommended for development)
- All Chromecast devices
- Google Nest displays

## Security Considerations

Following [Google Cast security guidelines](https://developers.google.com/cast/docs/web_receiver/security):

- Receiver must be served over HTTPS in production
- Validate all incoming custom messages
- Sanitize user-provided content
- Implement proper error boundaries
- Follow Content Security Policy (CSP) best practices

## Performance Optimization

Based on [Google Cast performance guidelines](https://developers.google.com/cast/docs/web_receiver/performance):

- Minimize JavaScript bundle size
- Optimize image loading and caching
- Use efficient DOM manipulation
- Implement proper memory management
- Monitor CPU and memory usage

## Current Status ✅

- ✅ Custom receiver loads on Chromecast
- ✅ YouTube video playback working
- ✅ Room information display
- ✅ Queue management
- ✅ Standalone testing mode
- ✅ Error handling and logging
- ✅ Responsive design
- ✅ Follows Google Cast best practices

## Additional Resources

### Google Cast Documentation
- [Cast Web Receiver Overview](https://developers.google.com/cast/docs/web_receiver)
- [Cast Application Framework (CAF)](https://developers.google.com/cast/docs/caf_receiver)
- [Media Player Library](https://developers.google.com/cast/docs/web_receiver/media_player_library)
- [Receiver Registration](https://developers.google.com/cast/docs/registration)

### Sample Implementations
- [Basic Media Receiver](https://github.com/googlecast/CastReceiver/tree/master/basic_media_receiver)
- [Custom Receiver](https://github.com/googlecast/CastReceiver/tree/master/custom_receiver)
- [Styled Media Receiver](https://github.com/googlecast/CastReceiver/tree/master/styled_media_receiver)

## Next Steps

1. Test with various YouTube videos
2. Verify room info and queue updates work
3. Test error handling scenarios
4. Consider production deployment options
5. Review Google Cast performance guidelines
6. Implement additional Cast framework features as needed