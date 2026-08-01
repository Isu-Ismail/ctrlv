import { writable } from 'svelte/store';
import { historyStore } from './historyStore';

export interface WSState {
  roomId: string;
  relayUrl: string;
  isConnected: boolean;
  isConnecting: boolean;
  browserCount: number;
  pcCount: number;
  cachedScreenshot: string | null;
  cachedPCText: string;
  webText: string;
  autoDownload: boolean;
  autoSolve: boolean;
  autoPush: boolean;
  errorModal: {
    isOpen: boolean;
    title: string;
    body: string;
  };
}

const getStoredRoomId = (): string => {
  if (typeof window === 'undefined') return 'ctrlv-a8f3b2';
  return localStorage.getItem('ctrlv_room_id') || 'ctrlv-a8f3b2';
};

const getStoredRelayUrl = (): string => {
  if (typeof window === 'undefined') return 'wss://ctrlv.onrender.com/ws';
  return localStorage.getItem('ctrlv_relay_url') || 'wss://ctrlv.onrender.com/ws';
};

const getStoredAutoDownload = (): boolean => {
  if (typeof window === 'undefined') return false;
  return localStorage.getItem('ctrlv_auto_download') === 'true';
};

const getStoredAutoSolve = (): boolean => {
  if (typeof window === 'undefined') return true;
  return localStorage.getItem('ctrlv_auto_solve') !== 'false';
};

const getStoredAutoPush = (): boolean => {
  if (typeof window === 'undefined') return true;
  return localStorage.getItem('ctrlv_auto_push') !== 'false';
};

const initialRoomId = getStoredRoomId();

const getInitialScreenshot = (roomId: string): string | null => {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(`ctrlv_last_screenshot_${roomId}`) || null;
};

const getInitialPCText = (roomId: string): string => {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem(`ctrlv_last_pc_text_${roomId}`) || '';
};

const getInitialWebText = (roomId: string): string => {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem(`ctrlv_last_web_text_${roomId}`) || '';
};

export const wsStore = writable<WSState>({
  roomId: initialRoomId,
  relayUrl: getStoredRelayUrl(),
  isConnected: false,
  isConnecting: false,
  browserCount: 0,
  pcCount: 0,
  cachedScreenshot: getInitialScreenshot(initialRoomId),
  cachedPCText: getInitialPCText(initialRoomId),
  webText: getInitialWebText(initialRoomId),
  autoDownload: getStoredAutoDownload(),
  autoSolve: getStoredAutoSolve(),
  autoPush: getStoredAutoPush(),
  errorModal: {
    isOpen: false,
    title: '',
    body: ''
  }
});

export function setRoomId(newRoomId: string) {
  if (!newRoomId || !newRoomId.trim()) return;
  const cleanId = newRoomId.trim();
  wsStore.update((s) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('ctrlv_room_id', cleanId);
    }
    return {
      ...s,
      roomId: cleanId,
      cachedScreenshot: getInitialScreenshot(cleanId),
      cachedPCText: getInitialPCText(cleanId),
      webText: getInitialWebText(cleanId)
    };
  });
  historyStore.setRoom(cleanId);
}

export function setRelayUrl(newUrl: string) {
  if (!newUrl || !newUrl.trim()) return;
  const cleanUrl = newUrl.trim();
  wsStore.update((s) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('ctrlv_relay_url', cleanUrl);
    }
    return { ...s, relayUrl: cleanUrl };
  });
}

export function setAutoDownload(enabled: boolean) {
  wsStore.update((s) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('ctrlv_auto_download', enabled ? 'true' : 'false');
    }
    return { ...s, autoDownload: enabled };
  });
}

export function setAutoSolve(enabled: boolean) {
  wsStore.update((s) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('ctrlv_auto_solve', enabled ? 'true' : 'false');
    }
    return { ...s, autoSolve: enabled };
  });
}

export function setAutoPush(enabled: boolean) {
  wsStore.update((s) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('ctrlv_auto_push', enabled ? 'true' : 'false');
    }
    return { ...s, autoPush: enabled };
  });
}

export function setCachedScreenshot(b64Data: string, roomId: string) {
  wsStore.update((s) => {
    if (typeof window !== 'undefined') {
      try {
        localStorage.setItem(`ctrlv_last_screenshot_${roomId}`, b64Data);
      } catch (e) {}
    }
    return { ...s, cachedScreenshot: b64Data };
  });
}

export function setCachedPCText(text: string, roomId: string) {
  wsStore.update((s) => {
    if (typeof window !== 'undefined') {
      try {
        localStorage.setItem(`ctrlv_last_pc_text_${roomId}`, text);
      } catch (e) {}
    }
    return { ...s, cachedPCText: text };
  });
}

export function setWebText(text: string, roomId: string) {
  wsStore.update((s) => {
    if (typeof window !== 'undefined') {
      try {
        localStorage.setItem(`ctrlv_last_web_text_${roomId}`, text);
      } catch (e) {}
    }
    return { ...s, webText: text };
  });
}

export function showErrorModal(title: string, body: string) {
  wsStore.update((s) => ({
    ...s,
    errorModal: { isOpen: true, title, body }
  }));
}

export function hideErrorModal() {
  wsStore.update((s) => ({
    ...s,
    errorModal: { ...s.errorModal, isOpen: false }
  }));
}
