import { writable } from 'svelte/store';

export type Theme = 'dark' | 'light';

const initialTheme: Theme = (typeof window !== 'undefined' && localStorage.getItem('ctrlv_theme') as Theme) || 'dark';

export const themeStore = writable<Theme>(initialTheme);

themeStore.subscribe((value) => {
  if (typeof window !== 'undefined') {
    localStorage.setItem('ctrlv_theme', value);
    document.documentElement.setAttribute('data-theme', value);
    if (value === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }
});

export function toggleTheme() {
  themeStore.update((curr) => (curr === 'dark' ? 'light' : 'dark'));
}
