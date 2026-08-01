import { get } from 'svelte/store';
import {
  wsStore,
  setCachedScreenshot,
  setCachedPCText,
  setWebText,
  showErrorModal
} from '../stores/wsStore';
import { historyStore } from '../stores/historyStore';
import { solveImageWithAI } from './aiService';
import type { WSIncomingMessage } from '../types/ws';

let socket: WebSocket | null = null;

// Track solved items to prevent duplicate auto-solving on reconnect/reload
const solvedItems = new Set<string>();

// Track downloaded items to prevent duplicate auto-downloads on reconnect/reload
const downloadedScreenshots = new Set<string>();

export function markAsSolved(content: string) {
  if (content) {
    solvedItems.add(content);
  }
}

export function isAlreadySolved(content: string): boolean {
  if (!content) return false;
  return solvedItems.has(content);
}

function buildWebSocketUrl(baseUrl: string, roomId: string): string {
  try {
    const url = new URL(baseUrl);
    url.searchParams.set('room', roomId);
    url.searchParams.set('client', 'browser');
    return url.toString();
  } catch (e) {
    const delimiter = baseUrl.includes('?') ? '&' : '?';
    return `${baseUrl}${delimiter}room=${encodeURIComponent(roomId)}&client=browser`;
  }
}

export function connectWebSocket() {
  const currentState = get(wsStore);

  if (socket) {
    if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
      return;
    }
    socket.onopen = null;
    socket.onmessage = null;
    socket.onerror = null;
    socket.onclose = null;
    socket = null;
  }

  // Pre-mark current cached screenshot or PC text so reconnecting doesn't auto-solve or auto-download existing content
  if (currentState.cachedScreenshot) {
    solvedItems.add(currentState.cachedScreenshot);
    downloadedScreenshots.add(currentState.cachedScreenshot);
  }
  if (currentState.cachedPCText) {
    solvedItems.add(currentState.cachedPCText);
  }

  wsStore.update((s) => ({ ...s, isConnecting: true }));

  const wsUrl = buildWebSocketUrl(currentState.relayUrl, currentState.roomId);

  try {
    socket = new WebSocket(wsUrl);

    socket.onopen = () => {
      wsStore.update((s) => ({ ...s, isConnected: true, isConnecting: false }));
      console.log('WebSocket Connected to:', wsUrl);

      // Send Join Room Payload for compatibility with JSON-based join handlers
      const joinMsg = JSON.stringify({
        type: 'join',
        room_id: currentState.roomId,
        client_type: 'browser'
      });
      socket?.send(joinMsg);
    };

    socket.onmessage = (event) => {
      try {
        const msg: WSIncomingMessage = JSON.parse(event.data);
        handleWSMessage(msg);
      } catch (e) {
        console.warn('Malformed WS message received:', event.data);
      }
    };

    socket.onerror = (err) => {
      console.error('WebSocket Error:', err);
    };

    socket.onclose = () => {
      wsStore.update((s) => ({ ...s, isConnected: false, isConnecting: false, browserCount: 0, pcCount: 0 }));
      console.log('WebSocket Disconnected.');
    };
  } catch (e) {
    console.error('Failed to create WebSocket:', e);
    wsStore.update((s) => ({ ...s, isConnected: false, isConnecting: false, browserCount: 0, pcCount: 0 }));
  }
}

export function disconnectWebSocket() {
  if (socket) {
    socket.onopen = null;
    socket.onmessage = null;
    socket.onerror = null;
    socket.onclose = null;
    try {
      socket.close();
    } catch (e) {}
    socket = null;
  }
  wsStore.update((s) => ({ ...s, isConnected: false, isConnecting: false, browserCount: 0, pcCount: 0 }));
}

export function sendTextToPC(text: string) {
  if (!text || !text.trim()) return;
  const state = get(wsStore);
  const cleanText = text.trim();

  // Save to history list with source='web_exe' (Web -> PC text)
  historyStore.addItem(cleanText, 'web_exe');

  if (!socket || socket.readyState !== WebSocket.OPEN) {
    showErrorModal('Connection Error', 'WebSocket is not connected. Please click Connect to join a room first.');
    return;
  }

  // Send payload supporting both web_exe and text message structures
  const payload = JSON.stringify({
    type: 'web_exe',
    room_id: state.roomId,
    content: cleanText,
    text: cleanText,
    client_type: 'browser'
  });

  socket.send(payload);
}

function handleWSMessage(msg: WSIncomingMessage) {
  const state = get(wsStore);

  // Connection Stats & Counts Broadcast (room_stats, joined, status, peer_count, counts)
  if (
    msg.type === 'room_stats' ||
    msg.type === 'joined' ||
    msg.type === 'status' ||
    msg.type === 'peer_count' ||
    msg.type === 'counts'
  ) {
    const browserNum =
      typeof msg.browsers === 'number'
        ? msg.browsers
        : typeof msg.browser === 'number'
        ? msg.browser
        : 0;

    const pcNum =
      typeof msg.clis === 'number'
        ? msg.clis
        : typeof msg.pc === 'number'
        ? msg.pc
        : typeof msg.cli === 'number'
        ? msg.cli
        : 0;

    wsStore.update((s) => ({
      ...s,
      browserCount: browserNum,
      pcCount: pcNum
    }));
  }
  // Image (Screenshot from PC)
  else if (msg.type === 'image') {
    const rawImage = msg.content || msg.data;
    if (rawImage) {
      let b64 = rawImage;
      if (!b64.startsWith('data:')) {
        b64 = 'data:image/jpeg;base64,' + b64;
      }
      setCachedScreenshot(b64, state.roomId);

      // Auto-save screenshot to downloads if enabled AND not already downloaded
      if (state.autoDownload && !downloadedScreenshots.has(b64)) {
        downloadedScreenshots.add(b64);
        triggerImageDownload(b64, `ctrlv_screenshot_${state.roomId}_${Date.now()}.jpg`);
      }

      // Auto-solve with AI if enabled AND this new screenshot has not already been solved
      if (state.autoSolve && !solvedItems.has(b64)) {
        solvedItems.add(b64);
        solveImageWithAI(b64);
      }
    }
  }
  // PC Sent Text to Browser (type: "exe_web" or "text")
  else if (msg.type === 'exe_web' || msg.type === 'text') {
    const incomingText = msg.content || msg.text;
    if (incomingText) {
      setCachedPCText(incomingText, state.roomId);
      historyStore.addItem(incomingText, 'exe_web');

      // Auto-solve with AI if enabled AND this text has not already been solved
      if (state.autoSolve && !solvedItems.has(incomingText)) {
        solvedItems.add(incomingText);
        solveImageWithAI(null, incomingText);
      }
    }
  }
  // Web Sent Text Echo (type: "web_exe")
  else if (msg.type === 'web_exe') {
    const incomingText = msg.content || msg.text;
    if (incomingText) {
      setWebText(incomingText, state.roomId);
    }
  }
}

export function triggerImageDownload(b64Data: string, filename: string) {
  if (!b64Data) return;
  downloadedScreenshots.add(b64Data);
  const a = document.createElement('a');
  a.href = b64Data;
  a.download = filename || `ctrlv_screenshot_${Date.now()}.jpg`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}
