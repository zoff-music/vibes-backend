# Vibez Mobile App 📱

A cross-platform mobile application built with React Native and Expo for the Vibez ecosystem.

## ✨ Features

- **Join Rooms**: Easily join existing rooms via name or QR code
- **Search & Queue**: Add songs from YouTube, Spotify, and SoundCloud
- **Cast Integration**: Integrated Google Cast support for casting to the big screen
- **Real-time Sync**: Synchronized room state and queue updates
- **Native Look & Feel**: Built with NativeWind (Tailwind CSS) for a modern, responsive UI

## 🚀 Getting Started

### Prerequisites

- pnpm installed
- [Expo Go](https://expo.dev/go) app on your mobile device (for local testing)
- Or an Android/iOS emulator

### Development

```bash
# Install dependencies
pnpm install

# Start Expo development server
pnpm start

# Clear cache and start
pnpm clean
```

### Platform Specifics

```bash
# Run on iOS emulator
pnpm ios

# Run on Android emulator
pnpm android

# Run in web browser
pnpm web
```

## 🛠 Tech Stack

- **Framework**: [Expo](https://expo.dev/) (SDK 54)
- **Engine**: [React Native](https://reactnative.dev/) (0.81.5)
- **Styling**: [NativeWind](https://www.nativewind.dev/) (Tailwind CSS v4)
- **Navigation**: [Expo Router](https://docs.expo.dev/router/introduction/)
- **Casting**: `react-native-google-cast`
- **Component Library**: Shared components from `@vibez/ui`
- **State Management**: Shared Zustand stores from `@vibez/shared`

## 📚 Rules

Follow the non-negotiable frontend coding conventions specified in [AGENTS.md](../../AGENTS.md).
