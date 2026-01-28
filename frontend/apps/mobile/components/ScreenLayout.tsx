import { LinearGradient } from 'expo-linear-gradient';
import React from 'react';
import { StatusBar, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

interface ScreenLayoutProps {
  children: React.ReactNode;
  safeArea?: boolean;
  className?: string; // For passing tailwind classes to the container
}

/**
 * ScreenLayout - Reusable Synthwave Background Container
 * Includes:
 * 1. Synth Sky Gradient
 * 2. Sun Hero (Glowing Orb)
 * 3. Retro Grid (Gradient Illusion)
 * 4. Scanlines Overlay
 */
export function ScreenLayout({
  children,
  safeArea = true,
  className = '',
}: ScreenLayoutProps) {
  return (
    <View style={styles.container} className={className}>
      <StatusBar barStyle="light-content" />

      {/* 1. Synth Sky Gradient */}
      <LinearGradient
        colors={['#140b2b', '#2a0f4e', '#6b1f62']}
        start={{ x: 0.5, y: 0 }}
        end={{ x: 0.5, y: 0.6 }}
        style={styles.sky}
      />

      {/* 2. Sun Hero (Sunset Orb) */}
      <View style={styles.sunHero} />

      {/* 3. Grid Floor (Simulated with Gradient) */}
      <LinearGradient
        colors={['transparent', 'rgba(255, 46, 151, 0.15)']}
        style={styles.gridFloor}
      />

      {/* Content */}
      {safeArea ? (
        <SafeAreaView style={styles.content}>{children}</SafeAreaView>
      ) : (
        <View style={styles.content}>{children}</View>
      )}

      {/* 4. Scanline Overlay */}
      <View style={styles.scanlines} pointerEvents="none">
        <LinearGradient
          colors={['rgba(0,0,0,0)', 'rgba(0,0,0,0.3)']}
          style={{ width: '100%', height: 4 }}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0d0d0f',
  },
  sky: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    height: '60%',
  },
  sunHero: {
    position: 'absolute',
    bottom: '15%',
    left: -100,
    right: -100,
    height: 300,
    backgroundColor: '#ff2e97', // Primary Pink
    borderRadius: 1000,
    opacity: 0.15,
    transform: [{ scaleX: 1.5 }],
    shadowColor: '#ff2e97',
    shadowOpacity: 0.8,
    shadowRadius: 100,
    shadowOffset: { width: 0, height: 0 },
  },
  gridFloor: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    height: '40%',
  },
  content: {
    flex: 1,
  },
  scanlines: {
    position: 'absolute',
    inset: 0,
    backgroundColor: 'transparent',
    zIndex: 100,
    opacity: 0.1,
    overflow: 'hidden',
  },
});
