import { get } from 'svelte/store';
import { aiConfigStore, aiSolverStatusStore } from '../stores/aiStore';
import { wsStore, setWebText, showErrorModal } from '../stores/wsStore';
import { sendTextToPC, markAsSolved } from './wsService';
import type { AIProvider } from '../types/ai';

let isSolving = false;

function resolveAIProvider(provider: AIProvider, apiKey: string): AIProvider {
  if (provider && provider !== 'auto') return provider;
  if (apiKey.startsWith('gsk_')) return 'groq';
  if (apiKey.startsWith('AIza')) return 'google';
  return 'openrouter';
}

function cleanMarkdownCodeBlocks(text: string): string {
  if (!text) return '';
  let cleaned = text.trim();
  // Strip ```lang ... ``` blocks if present
  if (cleaned.startsWith('```')) {
    cleaned = cleaned.replace(/^```[a-zA-Z]*\n?/, '').replace(/\n?```$/, '');
  }
  return cleaned.trim();
}

export async function solveImageWithAI(b64ImageData?: string | null, textPrompt?: string | null) {
  if (isSolving) return;

  const aiConfig = get(aiConfigStore);
  const wsState = get(wsStore);

  if (!aiConfig || !aiConfig.apiKey) {
    aiSolverStatusStore.set({ state: 'error', message: 'Configure AI Key in Config first!' });
    showErrorModal('AI Key Missing', 'Please open the Config tab and enter your API Key (e.g. from OpenRouter, Groq, or Google AI Studio).');
    return;
  }

  const cleanApiKey = aiConfig.apiKey.replace(/["'\s]/g, '');
  if (!cleanApiKey || cleanApiKey.length < 5) {
    aiSolverStatusStore.set({ state: 'error', message: 'AI API Key is invalid!' });
    showErrorModal('Invalid API Key', 'The provided API Key is invalid or empty. Please check your settings in the Config tab.');
    return;
  }

  const imageToSolve = b64ImageData || wsState.cachedScreenshot;
  const questionText = textPrompt || wsState.cachedPCText;

  if (!imageToSolve && !questionText) {
    aiSolverStatusStore.set({ state: 'error', message: 'No screenshot or text to solve' });
    return;
  }

  // Pre-mark item as solved so duplicate requests are not fired automatically
  if (imageToSolve) markAsSolved(imageToSolve);
  if (questionText) markAsSolved(questionText);

  isSolving = true;
  const provider = resolveAIProvider(aiConfig.provider, cleanApiKey);
  aiSolverStatusStore.set({ state: 'solving', message: `${provider.toUpperCase()} Analyzing...` });

  const prompt = aiConfig.customPrompt || 'Solve the problem shown. Output ONLY clean, working code without markdown explanations.';

  try {
    let generatedText = '';

    if (provider === 'openrouter') {
      const model = aiConfig.model || 'openrouter/auto';
      const content: any[] = [{ type: 'text', text: prompt }];

      if (questionText) {
        content.push({ type: 'text', text: `Context / Question Text: ${questionText}` });
      }
      if (imageToSolve) {
        let url = imageToSolve;
        if (!url.startsWith('data:')) url = 'data:image/jpeg;base64,' + url;
        content.push({ type: 'image_url', image_url: { url } });
      }

      const res = await fetch('https://openrouter.ai/api/v1/chat/completions', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${cleanApiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          model,
          max_tokens: aiConfig.maxTokens || 2048,
          messages: [{ role: 'user', content }]
        })
      });

      if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson?.error?.message || `OpenRouter Error ${res.status}`);
      }

      const resData = await res.json();
      generatedText = resData?.choices?.[0]?.message?.content || '';

    } else if (provider === 'groq') {
      const model = aiConfig.model || 'llama-3.2-11b-vision-preview';
      const content: any[] = [{ type: 'text', text: prompt }];

      if (questionText) {
        content.push({ type: 'text', text: `Context / Question Text: ${questionText}` });
      }
      if (imageToSolve) {
        let url = imageToSolve;
        if (!url.startsWith('data:')) url = 'data:image/jpeg;base64,' + url;
        content.push({ type: 'image_url', image_url: { url } });
      }

      const res = await fetch('https://api.groq.com/openai/v1/chat/completions', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${cleanApiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          model,
          messages: [{ role: 'user', content }]
        })
      });

      if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson?.error?.message || `Groq Error ${res.status}`);
      }

      const resData = await res.json();
      generatedText = resData?.choices?.[0]?.message?.content || '';

    } else if (provider === 'google') {
      const model = aiConfig.model || 'gemini-2.0-flash';
      const url = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent?key=${cleanApiKey}`;

      const parts: any[] = [{ text: prompt }];
      if (questionText) {
        parts.push({ text: `Question Text: ${questionText}` });
      }
      if (imageToSolve) {
        const base64Clean = imageToSolve.replace(/^data:image\/\w+;base64,/, '');
        parts.push({
          inline_data: {
            mime_type: 'image/jpeg',
            data: base64Clean
          }
        });
      }

      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          contents: [{ parts }]
        })
      });

      if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson?.error?.message || `Google AI Error ${res.status}`);
      }

      const resData = await res.json();
      generatedText = resData?.candidates?.[0]?.content?.parts?.[0]?.text || '';
    }

    if (aiConfig.codeOnly) {
      generatedText = cleanMarkdownCodeBlocks(generatedText);
    }

    if (generatedText) {
      setWebText(generatedText, wsState.roomId);
      if (wsState.autoPush) {
        sendTextToPC(generatedText);
        aiSolverStatusStore.set({ state: 'success', message: 'Solved & Pushed to PC!' });
      } else {
        aiSolverStatusStore.set({ state: 'success', message: 'Solved & Pasted Below!' });
      }

      setTimeout(() => {
        aiSolverStatusStore.set({ state: 'ready', message: 'AI Ready' });
      }, 3500);
    } else {
      throw new Error('AI Provider returned an empty response.');
    }
  } catch (err: any) {
    console.error('AI Solver Error:', err);
    aiSolverStatusStore.set({ state: 'error', message: 'AI Solver Error' });
    showErrorModal('AI Solver Error', err?.message || 'An unexpected error occurred during AI processing.');
  } finally {
    isSolving = false;
  }
}
