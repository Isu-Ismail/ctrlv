<script lang="ts">
  export let promptSymbol: string = '$';
  export let codeText: string = '';

  let copied = false;

  async function copyToClipboard() {
    if (!codeText) return;
    try {
      await navigator.clipboard.writeText(codeText);
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 2000);
    } catch (e) {
      console.error('Failed to copy code:', e);
    }
  }
</script>

<div class="relative flex items-center justify-between gap-3 bg-[var(--code-bg)] text-[var(--code-text)] border border-[var(--code-border)] rounded-xl py-4 px-4 pr-16 font-mono text-xs my-2 group min-h-[58px] shadow-sm">
  <div class="flex items-center gap-2.5 overflow-x-auto min-w-0 flex-1 no-scrollbar">
    <span class="text-[var(--accent-primary)] font-bold select-none shrink-0">{promptSymbol}</span>
    <span class="whitespace-nowrap text-[var(--code-text)] select-all font-mono leading-normal">{codeText}</span>
  </div>

  <button
    on:click={copyToClipboard}
    class="absolute right-3 top-1/2 -translate-y-1/2 w-8 h-8 rounded-lg bg-[var(--bg-tertiary)] hover:bg-[var(--accent-primary)]/15 text-[var(--text-muted)] hover:text-[var(--accent-primary)] border border-[var(--card-border)] hover:border-[var(--accent-primary)]/30 flex items-center justify-center transition-all cursor-pointer shadow-sm shrink-0"
    title="Copy code to clipboard"
  >
    {#if copied}
      <i class="fa-solid fa-check text-emerald-400 text-sm"></i>
    {:else}
      <i class="fa-regular fa-copy text-sm"></i>
    {/if}
  </button>
</div>
