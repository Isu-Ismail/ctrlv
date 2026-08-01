export type AIProvider = 'auto' | 'openrouter' | 'groq' | 'google';

export interface AIConfig {
  provider: AIProvider;
  apiKey: string;
  model: string;
  maxTokens: number;
  codeOnly: boolean;
  customPrompt: string;
}

export type AISolverState = 'ready' | 'solving' | 'success' | 'error';

export interface AISolverStatus {
  state: AISolverState;
  message: string;
}
