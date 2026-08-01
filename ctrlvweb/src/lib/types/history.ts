export type HistorySource = 'web_exe' | 'exe_web';

export interface HistoryItem {
  id: string;
  text: string;
  time: string;
  date: string;
  source?: HistorySource; // 'web_exe' = web -> exe, 'exe_web' = exe -> web
}
