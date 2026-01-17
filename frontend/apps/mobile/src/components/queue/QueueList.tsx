import React from 'react';
import { FlatList, StyleSheet, View } from 'react-native';
import { Song } from '@vibez/shared';
import { QueueItem } from './QueueItem';
import { Text } from '../ui/Text';

interface Props {
    songs: Song[];
    onRemove?: (id: string) => void;
    isAdmin?: boolean;
}

export const QueueList: React.FC<Props> = ({ songs, onRemove, isAdmin }) => {
    if (songs.length === 0) {
        return (
            <View style={styles.empty}>
                <Text color="muted">The queue is empty. Add some vibes!</Text>
            </View>
        );
    }

    return (
        <FlatList
            data={songs}
            keyExtractor={(item) => item.id}
            renderItem={({ item }) => (
                <QueueItem
                    song={item}
                    onRemove={onRemove}
                    isAdmin={isAdmin}
                />
            )}
            contentContainerStyle={styles.list}
            showsVerticalScrollIndicator={false}
        />
    );
};

const styles = StyleSheet.create({
    list: {
        paddingBottom: 20,
    },
    empty: {
        padding: 40,
        alignItems: 'center',
        justifyContent: 'center',
    },
});
