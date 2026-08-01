<script lang="ts">
  import { wsStore, hideErrorModal } from '../lib/stores/wsStore';

  $: modal = $wsStore.errorModal;
</script>

{#if modal.isOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-md p-4 animate-fade-in"
    on:click|self={hideErrorModal}
    on:keydown={(e) => e.key === 'Escape' && hideErrorModal()}
    tabindex="-1"
    role="dialog"
  >
    <div class="glass-panel max-w-lg w-full p-6 space-y-4 border-rose-500/40 shadow-2xl animate-scale-up">
      <div class="flex items-center gap-3 text-rose-500 font-bold text-lg">
        <i class="fa-solid fa-triangle-exclamation text-2xl"></i>
        <span>{modal.title || 'Error Notice'}</span>
      </div>

      <div class="font-mono text-xs text-[var(--text-main)] bg-[var(--bg-input)] p-4 rounded-lg border border-[var(--card-border)] max-h-64 overflow-y-auto whitespace-pre-wrap break-words leading-relaxed">
        {modal.body}
      </div>

      <div class="flex justify-end items-center gap-3 pt-2">
        <button
          on:click={hideErrorModal}
          class="px-4 py-2 rounded-lg bg-[var(--bg-tertiary)] hover:bg-[var(--card-hover)] text-[var(--text-muted)] text-xs font-semibold border border-[var(--card-border)] transition-all cursor-pointer"
        >
          Close
        </button>
        <button
          on:click={hideErrorModal}
          class="px-5 py-2 rounded-lg bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold transition-all cursor-pointer"
        >
          OK
        </button>
      </div>
    </div>
  </div>
{/if}
