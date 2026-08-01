export type ClientType = 'browser' | 'pc';

export interface WSJoinPayload {
  type: 'join';
  room_id: string;
  client_type: ClientType;
}

export interface WSTextPayload {
  type: 'text' | 'exe_web' | 'web_exe';
  room_id?: string;
  text?: string;
  content?: string;
}

export interface WSImagePayload {
  type: 'image';
  room_id?: string;
  data?: string;
  content?: string;
}

export interface WSCountsPayload {
  type: 'counts' | 'room_stats' | 'joined' | 'status' | 'peer_count';
  browser?: number;
  browsers?: number;
  pc?: number;
  clis?: number;
  cli?: number;
}

export type WSIncomingMessage = Record<string, any>;
