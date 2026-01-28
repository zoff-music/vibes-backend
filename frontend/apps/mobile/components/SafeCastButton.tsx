import Constants, { ExecutionEnvironment } from 'expo-constants';
import React from 'react';
import { CastButton } from 'react-native-google-cast';

export function SafeCastButton(props: React.ComponentProps<typeof CastButton>) {
  // Check if running in Expo Go
  const isExpoGo =
    Constants.executionEnvironment === ExecutionEnvironment.StoreClient;

  if (isExpoGo) {
    // Cast icon shouldn't be visible if cast isn't available
    return null;
  }

  // In development builds or standalone apps, render the real CastButton
  // If we wanted to show a custom icon when not connected, we could do it here
  // but CastButton usually handles its own state.
  // For the "RSS" issue, if they are using FontAwesome elsewhere, they should use MaterialCommunityIcons 'cast'
  return <CastButton {...props} />;
}
