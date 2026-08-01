<script lang="ts">
  import { historyStore } from '../lib/stores/historyStore';
  import { wsStore } from '../lib/stores/wsStore';
  import { sendTextToPC } from '../lib/services/wsService';

  $: roomId = $wsStore.roomId;
  $: historyList = $historyStore;

  let filterType: 'all' | 'web_exe' | 'exe_web' = 'all';

  $: filteredHistory = historyList.filter((item) => {
    if (filterType === 'all') return true;
    const src = item.source || 'exe_web';
    return src === filterType;
  });

  let selectedIndex = 0;
  let isMobileDetailOpen = false;
  let copyToast = false;
  let sendToast = false;

  $: if (selectedIndex >= filteredHistory.length) {
    selectedIndex = Math.max(0, filteredHistory.length - 1);
  }

  $: selectedItem = filteredHistory[selectedIndex] || null;

  function handleSelectMobileItem(idx: number) {
    selectedIndex = idx;
    isMobileDetailOpen = true;
  }

  async function copyText() {
    if (!selectedItem) return;
    try {
      await navigator.clipboard.writeText(selectedItem.text);
      copyToast = true;
      setTimeout(() => (copyToast = false), 1800);
    } catch (e) {}
  }

  function handleSendToPC() {
    if (!selectedItem) return;
    sendTextToPC(selectedItem.text);
    sendToast = true;
    setTimeout(() => (sendToast = false), 1800);
  }

  function handleDeleteItem() {
    if (!selectedItem) return;
    historyStore.deleteItem(selectedItem.id);
    if (filteredHistory.length <= 1) {
      isMobileDetailOpen = false;
    }
  }

  function handleClearAll() {
    if (confirm('Are you sure you want to clear all history for this room?')) {
      historyStore.clearAll();
      selectedIndex = 0;
      isMobileDetailOpen = false;
    }
  }
</script>

<div class="max-w-[1400px] w-full mx-auto p-3 sm:p-4 space-y-6">
  <div class="glass-panel p-4 sm:p-6 space-y-4">
    <!-- Header Title Bar -->
    <div class="flex items-center justify-between flex-wrap gap-3 pb-4 border-b border-[var(--card-border)]">
      <div class="flex items-center gap-2.5 text-sm sm:text-base font-extrabold text-[var(--accent-primary)]">
        <i class="fa-solid fa-clock-rotate-left"></i>
        <span>Room History</span>
        <span class="text-xs font-mono font-bold text-[var(--accent-cyan)] bg-[var(--bg-tertiary)] px-2 py-0.5 rounded border border-[var(--card-border)]">
          {roomId}
        </span>
      </div>

      <div class="flex items-center gap-3 flex-wrap">
        <!-- Filter Tabs: All, web -> exe, exe -> web -->
        <div class="flex items-center gap-1 bg-[var(--bg-tertiary)] p-1 rounded-xl border border-[var(--card-border)] text-xs">
          <button
            on:click={() => { filterType = 'all'; selectedIndex = 0; }}
            class={`px-3 py-1.5 rounded-lg font-bold transition-all cursor-pointer flex items-center gap-1.5 ${
              filterType === 'all'
                ? 'bg-[var(--accent-primary)] text-white shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            <i class="fa-solid fa-list"></i>
            <span>All ({historyList.length})</span>
          </button>

          <button
            on:click={() => { filterType = 'web_exe'; selectedIndex = 0; }}
            class={`px-3 py-1.5 rounded-lg font-bold transition-all cursor-pointer flex items-center gap-1.5 ${
              filterType === 'web_exe'
                ? 'bg-cyan-600 text-white shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            <i class="fa-solid fa-globe text-cyan-300"></i>
            <span>web &rarr; exe</span>
          </button>

          <button
            on:click={() => { filterType = 'exe_web'; selectedIndex = 0; }}
            class={`px-3 py-1.5 rounded-lg font-bold transition-all cursor-pointer flex items-center gap-1.5 ${
              filterType === 'exe_web'
                ? 'bg-indigo-600 text-white shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            <i class="fa-solid fa-laptop text-indigo-300"></i>
            <span>exe &rarr; web</span>
          </button>
        </div>

        <button
          on:click={handleClearAll}
          disabled={historyList.length === 0}
          class="px-3.5 py-2 rounded-xl text-xs font-bold text-rose-400 border border-rose-500/30 hover:bg-rose-500/10 disabled:opacity-40 transition-all cursor-pointer flex items-center gap-1.5"
        >
          <i class="fa-solid fa-trash-can"></i>
          <span class="hidden sm:inline">Clear All</span>
        </button>
      </div>
    </div>

    {#if filteredHistory.length === 0}
      <div class="py-16 text-center space-y-3">
        <i class="fa-solid fa-folder-open text-5xl text-[var(--text-dim)] opacity-40"></i>
        <h3 class="font-bold text-base text-[var(--text-main)]">No History Items Found</h3>
        <p class="text-xs text-[var(--text-muted)] max-w-sm mx-auto">
          {#if filterType === 'all'}
            Texts sent between your Web browser and PC will automatically appear here.
          {:else if filterType === 'web_exe'}
            No texts sent from Web to PC (web &rarr; exe) recorded yet.
          {:else}
            No texts received from PC to Web (exe &rarr; web) recorded yet.
          {/if}
        </p>
      </div>
    {:else}
      <!-- MOBILE VIEW: Drill-down navigation (List OR Detail page) -->
      <div class="block md:hidden">
        {#if isMobileDetailOpen && selectedItem}
          <!-- Mobile Detail Page view -->
          <div class="bg-[var(--bg-tertiary)] border border-[var(--card-border)] rounded-xl p-4 flex flex-col gap-4 min-h-[440px] animate-fade-in">
            <!-- Top Mobile Navigation & Source Badge Bar -->
            <div class="flex items-center justify-between gap-2 pb-3 border-b border-[var(--card-border)]">
              <button
                on:click={() => (isMobileDetailOpen = false)}
                class="px-3 py-1.5 rounded-lg bg-[var(--bg-input)] text-xs font-bold text-[var(--accent-cyan)] border border-[var(--card-border)] flex items-center gap-1.5 cursor-pointer"
              >
                <i class="fa-solid fa-arrow-left"></i>
                <span>Back</span>
              </button>

              <div class="flex items-center gap-2">
                {#if selectedItem.source === 'web_exe'}
                  <span class="px-2 py-0.5 rounded text-[10px] font-extrabold bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 flex items-center gap-1">
                    <i class="fa-solid fa-globe"></i> web &rarr; exe
                  </span>
                {:else}
                  <span class="px-2 py-0.5 rounded text-[10px] font-extrabold bg-indigo-500/10 text-indigo-400 border border-indigo-500/30 flex items-center gap-1">
                    <i class="fa-solid fa-laptop"></i> exe &rarr; web
                  </span>
                {/if}
                <span class="text-[11px] font-semibold text-[var(--text-muted)]">
                  {selectedItem.time}
                </span>
              </div>
            </div>

            <!-- Mobile Actions Row -->
            <div class="flex items-center justify-between gap-2">
              <button
                on:click={copyText}
                class="flex-1 px-3 py-2 rounded-lg bg-[var(--bg-input)] text-xs font-bold text-[var(--text-main)] border border-[var(--card-border)] flex items-center justify-center gap-1.5 cursor-pointer"
              >
                {#if copyToast}
                  <i class="fa-solid fa-check text-emerald-400"></i>
                  <span>Copied</span>
                {:else}
                  <i class="fa-regular fa-copy"></i>
                  <span>Copy</span>
                {/if}
              </button>

              <button
                on:click={handleSendToPC}
                class="flex-1 px-3 py-2 rounded-lg bg-[var(--accent-primary)] text-white text-xs font-bold flex items-center justify-center gap-1.5 shadow cursor-pointer"
              >
                {#if sendToast}
                  <i class="fa-solid fa-circle-check text-emerald-300"></i>
                  <span>Sent!</span>
                {:else}
                  <i class="fa-solid fa-paper-plane"></i>
                  <span>Send to PC</span>
                {/if}
              </button>

              <button
                on:click={handleDeleteItem}
                class="px-3 py-2 rounded-lg text-rose-400 border border-rose-500/30 text-xs font-bold flex items-center justify-center cursor-pointer"
                title="Delete this entry"
              >
                <i class="fa-solid fa-trash-can"></i>
              </button>
            </div>

            <!-- Detail Text Display -->
            <div class="flex-1 min-h-[300px]">
              <textarea
                value={selectedItem.text}
                readonly
                class="w-full h-full min-h-[300px] p-3.5 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-lg text-xs font-mono text-[var(--text-main)] outline-none resize-none custom-scrollbar leading-relaxed"
              ></textarea>
            </div>
          </div>
        {:else}
          <!-- Mobile List View -->
          <div class="space-y-2">
            <p class="text-xs font-bold text-[var(--text-muted)] mb-2">Tap an item to view details &amp; actions:</p>
            {#each filteredHistory as item, idx (item.id)}
              <button
                type="button"
                on:click={() => handleSelectMobileItem(idx)}
                class="w-full text-left p-3.5 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--card-border)] hover:border-[var(--accent-primary)]/40 transition-all cursor-pointer flex items-center justify-between gap-3 group"
              >
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2 mb-1">
                    {#if item.source === 'web_exe'}
                      <span class="px-2 py-0.5 rounded text-[10px] font-extrabold bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 flex items-center gap-1 shrink-0">
                        <i class="fa-solid fa-globe"></i> web &rarr; exe
                      </span>
                    {:else}
                      <span class="px-2 py-0.5 rounded text-[10px] font-extrabold bg-indigo-500/10 text-indigo-400 border border-indigo-500/30 flex items-center gap-1 shrink-0">
                        <i class="fa-solid fa-laptop"></i> exe &rarr; web
                      </span>
                    {/if}
                    <span class="text-[11px] text-[var(--text-muted)]">
                      {item.time || 'Recent'}
                    </span>
                  </div>
                  <div class="text-xs font-bold text-[var(--text-main)] truncate">
                    {item.text.trim().split('\n')[0] || 'Empty Text'}
                  </div>
                </div>
                <i class="fa-solid fa-chevron-right text-xs text-[var(--text-muted)] group-hover:text-[var(--accent-primary)] transition-colors"></i>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- DESKTOP VIEW: Split 2-column Side-by-Side Layout -->
      <div class="hidden md:grid grid-cols-12 gap-4 min-h-[480px]">
        <!-- Sidebar Item List -->
        <div class="col-span-4 bg-[var(--bg-tertiary)] border border-[var(--card-border)] rounded-xl p-2 space-y-1.5 max-h-[560px] overflow-y-auto custom-scrollbar">
          {#each filteredHistory as item, idx (item.id)}
            <button
              type="button"
              on:click={() => (selectedIndex = idx)}
              class={`w-full text-left p-3 rounded-xl border transition-all cursor-pointer ${
                idx === selectedIndex
                  ? 'bg-[var(--card-bg)] border-[var(--accent-primary)] shadow-sm'
                  : 'bg-transparent border-transparent hover:bg-[var(--card-hover)]'
              }`}
            >
              <div class="flex items-center justify-between gap-1.5 mb-1.5">
                {#if item.source === 'web_exe'}
                  <span class="px-2 py-0.5 rounded text-[10px] font-extrabold bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 inline-flex items-center gap-1">
                    <i class="fa-solid fa-globe"></i> web &rarr; exe
                  </span>
                {:else}
                  <span class="px-2 py-0.5 rounded text-[10px] font-extrabold bg-indigo-500/10 text-indigo-400 border border-indigo-500/30 inline-flex items-center gap-1">
                    <i class="fa-solid fa-laptop"></i> exe &rarr; web
                  </span>
                {/if}

                <span class="text-[10px] text-[var(--text-muted)] font-mono">
                  {item.time}
                </span>
              </div>

              <div class="text-xs font-semibold text-[var(--text-main)] truncate">
                {item.text.trim().split('\n')[0] || 'Empty Text'}
              </div>
            </button>
          {/each}
        </div>

        <!-- Right Detail Panel -->
        <div class="col-span-8 bg-[var(--bg-tertiary)] border border-[var(--card-border)] rounded-xl p-4 flex flex-col gap-3 min-h-[480px]">
          {#if selectedItem}
            <div class="flex items-center justify-between flex-wrap gap-2 pb-2 border-b border-[var(--card-border)]">
              <div class="flex items-center gap-2.5 text-xs font-bold text-[var(--text-muted)]">
                {#if selectedItem.source === 'web_exe'}
                  <span class="px-2.5 py-1 rounded-md text-xs font-extrabold bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 inline-flex items-center gap-1.5">
                    <i class="fa-solid fa-globe"></i> web &rarr; exe
                  </span>
                {:else}
                  <span class="px-2.5 py-1 rounded-md text-xs font-extrabold bg-indigo-500/10 text-indigo-400 border border-indigo-500/30 inline-flex items-center gap-1.5">
                    <i class="fa-solid fa-laptop"></i> exe &rarr; web
                  </span>
                {/if}
                <span>Recorded: {selectedItem.time || 'Recent'} {selectedItem.date ? `(${selectedItem.date})` : ''}</span>
              </div>

              <div class="flex items-center gap-2">
                <button
                  on:click={copyText}
                  class="px-3 py-1.5 rounded-lg bg-[var(--bg-input)] hover:bg-[var(--card-hover)] text-xs font-semibold text-[var(--text-main)] border border-[var(--card-border)] flex items-center gap-1.5 transition-all cursor-pointer"
                >
                  {#if copyToast}
                    <i class="fa-solid fa-check text-emerald-400"></i>
                    <span>Copied</span>
                  {:else}
                    <i class="fa-regular fa-copy"></i>
                    <span>Copy Text</span>
                  {/if}
                </button>

                <button
                  on:click={handleSendToPC}
                  class="px-3 py-1.5 rounded-lg bg-[var(--accent-primary)] hover:bg-[var(--accent-hover)] text-white text-xs font-bold flex items-center gap-1.5 shadow transition-all cursor-pointer"
                >
                  {#if sendToast}
                    <i class="fa-solid fa-circle-check text-emerald-300"></i>
                    <span>Sent!</span>
                  {:else}
                    <i class="fa-solid fa-paper-plane"></i>
                    <span>Send to PC</span>
                  {/if}
                </button>

                <button
                  on:click={handleDeleteItem}
                  class="p-1.5 px-2.5 rounded-lg text-rose-400 hover:bg-rose-500/10 border border-rose-500/30 text-xs transition-all cursor-pointer"
                  title="Delete this entry"
                >
                  <i class="fa-solid fa-trash-can"></i>
                </button>
              </div>
            </div>

            <div class="flex-1 flex flex-col">
              <textarea
                value={selectedItem.text}
                readonly
                class="w-full flex-1 p-4 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-lg text-xs font-mono text-[var(--text-main)] outline-none resize-none custom-scrollbar leading-relaxed"
              ></textarea>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>
