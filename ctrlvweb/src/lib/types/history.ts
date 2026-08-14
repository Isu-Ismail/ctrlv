export type HistorySource = 'web_exe' | 'exe_web';

export interface HistoryItem {
  id: string;
  text: string;
  image?: string; // base64 image data if screenshot/camera photo was captured
  time: string;
  date: string;
  source?: HistorySource; // 'web_exe' = web -> exe, 'exe_web' = exe -> web
}
