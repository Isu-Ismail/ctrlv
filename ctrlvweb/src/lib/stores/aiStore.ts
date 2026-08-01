import { writable, derived } from 'svelte/store';
import type { AIConfig, AISolverStatus } from '../types/ai';

const DEFAULT_PROMPT = 'Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting.';

function getInitialAIConfig(): AIConfig {
  if (typeof window === 'undefined') {
    return {
      provider: 'openrouter',
      apiKey: '',
      model: 'openrouter/auto',
      maxTokens: 2048,
      codeOnly: true,
      customPrompt: DEFAULT_PROMPT
    };
  }

  try {
    const stored = localStorage.getItem('ctrlv_ai_config');
    if (stored) {
      const parsed = JSON.parse(stored);
      return {
        provider: parsed.provider || 'openrouter',
        apiKey: (parsed.apiKey || '').trim(),
        model: parsed.model || 'openrouter/auto',
        maxTokens: parseInt(parsed.maxTokens) || 2048,
        codeOnly: parsed.codeOnly !== false,
        customPrompt: parsed.customPrompt || DEFAULT_PROMPT
      };
    }
  } catch (e) {
    console.warn('Failed to load AI config from localStorage:', e);
  }

  return {
    provider: 'openrouter',
    apiKey: '',
    model: 'openrouter/auto',
    maxTokens: 2048,
    codeOnly: true,
    customPrompt: DEFAULT_PROMPT
  };
}

export const aiConfigStore = writable<AIConfig>(getInitialAIConfig());

aiConfigStore.subscribe((config) => {
  if (typeof window !== 'undefined') {
    try {
      localStorage.setItem('ctrlv_ai_config', JSON.stringify(config));
    } catch (e) {
      console.warn('Failed to save AI config to localStorage:', e);
    }
  }
});

export const aiSolverStatusStore = writable<AISolverStatus>({
  state: 'ready',
  message: 'AI Ready'
});

export function saveAIConfig(newConfig: Partial<AIConfig>) {
  aiConfigStore.update((curr) => ({
    ...curr,
    ...newConfig
  }));
}

export function resetAIConfig() {
  const defaultConfig: AIConfig = {
    provider: 'openrouter',
    apiKey: '',
    model: 'openrouter/auto',
    maxTokens: 2048,
    codeOnly: true,
    customPrompt: DEFAULT_PROMPT
  };
  aiConfigStore.set(defaultConfig);
}
