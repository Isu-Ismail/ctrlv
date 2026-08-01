import { writable } from 'svelte/store';
import type { HistoryItem, HistorySource } from '../types/history';

function loadHistoryFromStorage(roomId: string): HistoryItem[] {
  if (typeof window === 'undefined') return [];
  try {
    const key = `ctrlv_history_${roomId}`;
    const stored = localStorage.getItem(key);
    if (!stored) return [];
    const parsed: HistoryItem[] = JSON.parse(stored);
    return parsed.map((item) => {
      // Backwards compatibility migration
      let src: HistorySource = 'exe_web';
      if (item.source === 'web_exe' || item.source === ('web' as any)) {
        src = 'web_exe';
      } else if (item.source === 'exe_web' || item.source === ('pc' as any)) {
        src = 'exe_web';
      }
      return { ...item, source: src };
    });
  } catch (e) {
    return [];
  }
}

export function createHistoryStore(initialRoomId: string) {
  const { subscribe, set, update } = writable<HistoryItem[]>(loadHistoryFromStorage(initialRoomId));

  let currentRoom = initialRoomId;

  return {
    subscribe,
    setRoom: (roomId: string) => {
      currentRoom = roomId;
      set(loadHistoryFromStorage(roomId));
    },
    addItem: (text: string, source: HistorySource = 'exe_web') => {
      if (!text || !text.trim()) return;
      const cleanText = text.trim();
      update((list) => {
        // Prevent immediate duplicate entries with same source
        if (list.length > 0 && list[0].text.trim() === cleanText && (list[0].source || 'exe_web') === source) {
          return list;
        }
        const now = new Date();
        const newItem: HistoryItem = {
          id: Date.now().toString(36) + Math.random().toString(36).substring(2),
          text: cleanText,
          time: now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          date: now.toLocaleDateString(),
          source: source
        };
        const updated = [newItem, ...list];
        if (typeof window !== 'undefined') {
          try {
            localStorage.setItem(`ctrlv_history_${currentRoom}`, JSON.stringify(updated));
          } catch (e) {}
        }
        return updated;
      });
    },
    deleteItem: (id: string) => {
      update((list) => {
        const updated = list.filter((item) => item.id !== id);
        if (typeof window !== 'undefined') {
          try {
            localStorage.setItem(`ctrlv_history_${currentRoom}`, JSON.stringify(updated));
          } catch (e) {}
        }
        return updated;
      });
    },
    clearAll: () => {
      set([]);
      if (typeof window !== 'undefined') {
        localStorage.removeItem(`ctrlv_history_${currentRoom}`);
      }
    }
  };
}

export const historyStore = createHistoryStore('ctrlv-a8f3b2');
