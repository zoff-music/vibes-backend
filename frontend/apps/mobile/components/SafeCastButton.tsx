import FontAwesome from '@expo/vector-icons/FontAwesome';
import Constants, { ExecutionEnvironment } from 'expo-constants';
import React from 'react';
import { TouchableOpacity } from 'react-native';
import { CastButton } from 'react-native-google-cast';

export function SafeCastButton(props: React.ComponentProps<typeof CastButton>) {
  // Check if running in Expo Go
  const isExpoGo =
    Constants.executionEnvironment === ExecutionEnvironment.StoreClient;

  if (isExpoGo) {
    return (
      <TouchableOpacity
        style={props.style}
        onPress={() =>
          alert(
            'Chromecast is only available in development builds, not Expo Go.',
          )
        }
        className="items-center justify-center opacity-50"
      >
        <FontAwesome name="rss" size={20} color="white" />
      </TouchableOpacity>
    );
  }

  // In development builds or standalone apps, render the real CastButton
  return <CastButton {...props} />;
}
