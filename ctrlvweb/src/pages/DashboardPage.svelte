<script lang="ts">
  import { setActiveTab } from '../lib/stores/viewStore';
  import {
    wsStore,
    setAutoDownload,
    setAutoSolve,
    setAutoPush,
    setWebText
  } from '../lib/stores/wsStore';
  import { aiConfigStore, aiSolverStatusStore } from '../lib/stores/aiStore';
  import { sendTextToPC, triggerImageDownload } from '../lib/services/wsService';
  import { solveImageWithAI } from '../lib/services/aiService';

  $: screenshot = $wsStore.cachedScreenshot;
  $: pcText = $wsStore.cachedPCText;
  $: webText = $wsStore.webText;
  $: roomId = $wsStore.roomId;
  $: autoDownload = $wsStore.autoDownload;
  $: autoSolve = $wsStore.autoSolve;
  $: autoPush = $wsStore.autoPush;

  $: aiConfig = $aiConfigStore;
  $: aiStatus = $aiSolverStatusStore;

  let activeTab: 'screenshot' | 'pctext' = 'screenshot';
  let isImageModalOpen = false;
  let copiedTextToast = false;
  let sentTextToast = false;

  // Send limit and locking state
  let lastSentText = '';
  let consecutiveSendCount = 0;
  let isSendingToPC = false;

  function handleWebTextChange(e: Event) {
    const val = (e.target as HTMLTextAreaElement).value;
    setWebText(val, roomId);
    if (val.trim() !== lastSentText) {
      consecutiveSendCount = 0;
    }
  }

  function handleSendToPC() {
    if (!webText || !webText.trim() || isSendingToPC) return;

    const currentText = webText.trim();
    if (currentText === lastSentText) {
      if (consecutiveSendCount >= 3) {
        return; // Block send after 3 consecutive identical sends
      }
      consecutiveSendCount++;
    } else {
      lastSentText = currentText;
      consecutiveSendCount = 1;
    }

    isSendingToPC = true;
    sendTextToPC(currentText);

    sentTextToast = true;
    setTimeout(() => {
      sentTextToast = false;
      isSendingToPC = false;
    }, 1200);
  }

  async function copyPCText() {
    if (!pcText) return;
    try {
      await navigator.clipboard.writeText(pcText);
      copiedTextToast = true;
      setTimeout(() => (copiedTextToast = false), 1800);
    } catch (e) {}
  }

  async function handleSolveClick() {
    if (activeTab === 'screenshot' && screenshot) {
      await solveImageWithAI(screenshot);
    } else if (pcText) {
      await solveImageWithAI(null, pcText);
    }
  }

  function handleDownloadClick() {
    if (screenshot) {
      triggerImageDownload(screenshot, `ctrlv_screenshot_${roomId}_${Date.now()}.jpg`);
    }
  }
</script>

<!-- PC View: Fits perfectly in viewport (md:h-[calc(100vh-156px)]) with equal top & bottom padding. Mobile View: Normal scrollable -->
<div class="max-w-[1400px] w-full mx-auto p-4 sm:p-5 md:h-[calc(100vh-156px)] md:overflow-hidden flex flex-col justify-between my-auto">
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 flex-1 min-h-0 h-full">

    <!-- Left Column: Screenshot / PC Text Sync -->
    <div class="glass-panel p-4 sm:p-5 flex flex-col gap-3.5 h-full min-h-0">
      <div class="flex items-center justify-between flex-wrap gap-2.5 pb-2.5 border-b border-[var(--card-border)] shrink-0">
        <!-- Switcher Tabs -->
        <div class="flex items-center gap-1 bg-[var(--bg-tertiary)] p-1 rounded-lg border border-[var(--card-border)] w-full sm:w-auto">
          <button
            on:click={() => (activeTab = 'screenshot')}
            class={`flex-1 sm:flex-none px-3.5 py-1.5 rounded-md text-xs sm:text-sm font-bold flex items-center justify-center gap-2 transition-all cursor-pointer ${
              activeTab === 'screenshot'
                ? 'bg-[var(--accent-primary)] text-white shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            <i class="fa-solid fa-desktop"></i>
            <span>Screenshot</span>
          </button>
          <button
            on:click={() => (activeTab = 'pctext')}
            class={`flex-1 sm:flex-none px-3.5 py-1.5 rounded-md text-xs sm:text-sm font-bold flex items-center justify-center gap-2 transition-all cursor-pointer ${
              activeTab === 'pctext'
                ? 'bg-[var(--accent-primary)] text-white shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            <i class="fa-solid fa-keyboard"></i>
            <span>PC Text</span>
          </button>
        </div>

        <!-- Quick Toggles: Save & Auto (Left Column) -->
        <div class="flex items-center gap-1.5 w-full sm:w-auto flex-wrap">
          <button
            on:click={() => setAutoDownload(!autoDownload)}
            class={`flex-1 sm:flex-none justify-center px-2.5 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 border transition-all cursor-pointer ${
              autoDownload
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--card-border)]'
            }`}
            title="Auto-save incoming screenshots to browser downloads"
          >
            <i class="fa-solid fa-floppy-disk"></i>
            <span>Save: {autoDownload ? 'ON' : 'OFF'}</span>
          </button>

          <button
            on:click={() => setAutoSolve(!autoSolve)}
            class={`flex-1 sm:flex-none justify-center px-2.5 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 border transition-all cursor-pointer ${
              autoSolve
                ? 'bg-amber-500/10 text-amber-400 border-amber-500/30'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--card-border)]'
            }`}
            title="Automatically solve incoming screenshots/text with AI"
          >
            <i class="fa-solid fa-wand-magic-sparkles"></i>
            <span>Auto: {autoSolve ? 'ON' : 'OFF'}</span>
          </button>
        </div>
      </div>

      <!-- Content Viewers -->
      {#if activeTab === 'screenshot'}
        <div class="relative flex-1 min-h-[260px] bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl flex items-center justify-center p-3 overflow-hidden group">
          {#if screenshot}
            <button
              type="button"
              on:click={() => (isImageModalOpen = true)}
              class="border-none bg-transparent p-0 flex items-center justify-center cursor-zoom-in group outline-none h-full w-full"
              title="Click to view full screen"
            >
              <img
                src={screenshot}
                alt="Latest Screenshot"
                class="max-h-full max-w-full object-contain rounded transition-transform duration-200 group-hover:scale-[1.01]"
              />
            </button>
          {:else}
            <div class="text-center space-y-3 py-10 px-4">
              <i class="fa-solid fa-image text-6xl text-[var(--text-dim)] opacity-40"></i>
              <div class="font-bold text-lg text-[var(--text-main)]">No Screenshot Received</div>
              <p class="text-xs sm:text-sm text-[var(--text-muted)] max-w-sm mx-auto">
                Press <kbd class="px-2.5 py-1 bg-[var(--bg-tertiary)] border border-[var(--card-border)] rounded text-xs font-mono text-[var(--accent-cyan)] font-bold">Ctrl + Shift + S</kbd> on your PC to sync instantly
              </p>
            </div>
          {/if}
        </div>
      {:else}
        <div class="relative flex-1 min-h-[260px] bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl flex flex-col p-3.5">
          <textarea
            value={pcText}
            readonly
            placeholder="Text synced from PC clipboard will appear here automatically..."
            class="w-full flex-1 bg-transparent border-none outline-none font-mono text-xs sm:text-sm text-[var(--text-main)] resize-none custom-scrollbar p-1 leading-relaxed"
          ></textarea>
        </div>
      {/if}

      <!-- Bottom Actions Bar -->
      <div class="flex items-center justify-between gap-3 pt-2 border-t border-[var(--card-border)] flex-wrap shrink-0">
        {#if activeTab === 'screenshot'}
          <button
            on:click={handleDownloadClick}
            disabled={!screenshot}
            class="flex-1 sm:flex-none px-4 py-2 rounded-xl bg-[var(--bg-tertiary)] hover:bg-[var(--card-hover)] disabled:opacity-40 text-xs sm:text-sm font-semibold text-[var(--text-main)] border border-[var(--card-border)] flex items-center justify-center gap-2 transition-all cursor-pointer"
          >
            <i class="fa-solid fa-download"></i>
            <span>Download Image</span>
          </button>
        {:else}
          <button
            on:click={copyPCText}
            disabled={!pcText}
            class="flex-1 sm:flex-none px-4 py-2 rounded-xl bg-[var(--bg-tertiary)] hover:bg-[var(--card-hover)] disabled:opacity-40 text-xs sm:text-sm font-semibold text-[var(--text-main)] border border-[var(--card-border)] flex items-center justify-center gap-2 transition-all cursor-pointer"
          >
            {#if copiedTextToast}
              <i class="fa-solid fa-check text-emerald-400"></i>
              <span>Copied!</span>
            {:else}
              <i class="fa-regular fa-copy"></i>
              <span>Copy Text</span>
            {/if}
          </button>
        {/if}

        <button
          on:click={handleSolveClick}
          class="flex-1 sm:flex-none px-5 py-2 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white font-bold text-xs sm:text-sm flex items-center justify-center gap-2 shadow-lg shadow-indigo-500/20 transition-all cursor-pointer sm:ml-auto"
        >
          <i class="fa-solid fa-wand-magic-sparkles"></i>
          <span>Solve with AI</span>
        </button>
      </div>
    </div>

    <!-- Right Column: AI Output & Live Text Sync Editor -->
    <div class="glass-panel p-4 sm:p-5 flex flex-col gap-3.5 h-full min-h-0">
      <div class="flex items-center justify-between flex-wrap gap-2 pb-2.5 border-b border-[var(--card-border)] shrink-0">
        <div class="flex items-center gap-2 font-bold text-sm sm:text-base text-[var(--text-main)]">
          <i class="fa-solid fa-code text-[var(--accent-primary)]"></i>
          <span>Snippet / AI Output</span>
        </div>

        <div class="flex items-center gap-2 flex-wrap">
          <!-- Push Toggle Button (Moved next to AI Ready status) -->
          <button
            on:click={() => setAutoPush(!autoPush)}
            class={`px-2.5 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 border transition-all cursor-pointer ${
              autoPush
                ? 'bg-cyan-500/10 text-cyan-400 border-cyan-500/30'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--card-border)]'
            }`}
            title="Auto-push AI solution directly to PC clipboard"
          >
            <i class="fa-solid fa-paper-plane"></i>
            <span>Push: {autoPush ? 'ON' : 'OFF'}</span>
          </button>

          <!-- AI Solver Status Pill -->
          <div
            class={`px-3 py-1 rounded-full text-xs sm:text-sm font-bold border flex items-center gap-1.5 ${
              aiStatus.state === 'solving'
                ? 'bg-blue-500/10 text-blue-400 border-blue-500/30 animate-pulse'
                : aiStatus.state === 'success'
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                : aiStatus.state === 'error'
                ? 'bg-rose-500/10 text-rose-400 border-rose-500/30'
                : 'bg-indigo-500/10 text-indigo-400 border-indigo-500/30'
            }`}
          >
            {#if aiStatus.state === 'solving'}
              <i class="fa-solid fa-spinner fa-spin"></i>
            {:else if aiStatus.state === 'success'}
              <i class="fa-solid fa-circle-check"></i>
            {:else if aiStatus.state === 'error'}
              <i class="fa-solid fa-triangle-exclamation"></i>
            {:else}
              <i class="fa-solid fa-robot"></i>
            {/if}
            <span>{aiStatus.message}</span>
          </div>
        </div>
      </div>

      <!-- Live Web Text Area -->
      <div class="relative flex-1 min-h-[260px] bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl flex flex-col p-3.5">
        <textarea
          value={webText}
          on:input={handleWebTextChange}
          placeholder="Type or paste text here to automatically sync with your PC clipboard in real-time..."
          class="w-full flex-1 bg-transparent border-none outline-none font-mono text-xs sm:text-sm text-[var(--text-main)] resize-none custom-scrollbar p-1 leading-relaxed"
        ></textarea>
      </div>

      <!-- Send to PC Action Footer (with send locking & 3-time max limit) -->
      <div class="flex items-center justify-between gap-3 pt-2 border-t border-[var(--card-border)] flex-wrap shrink-0">
        <span class="text-xs text-[var(--text-muted)] flex items-center gap-1.5">
          <i class="fa-solid fa-info-circle text-[var(--accent-cyan)]"></i>
          <span>Text auto-syncs to PC clipboard upon sending</span>
        </span>

        <button
          on:click={handleSendToPC}
          disabled={!webText || !webText.trim() || isSendingToPC || (webText.trim() === lastSentText && consecutiveSendCount >= 3)}
          class={`w-full sm:w-auto px-6 py-2 rounded-xl text-white font-extrabold text-xs sm:text-sm flex items-center justify-center gap-2 shadow-lg transition-all cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed ${
            webText.trim() === lastSentText && consecutiveSendCount >= 3
              ? 'bg-rose-600 hover:bg-rose-600 shadow-rose-600/20'
              : 'bg-[var(--accent-primary)] hover:bg-[var(--accent-hover)] shadow-indigo-500/25'
          }`}
        >
          {#if isSendingToPC}
            <i class="fa-solid fa-spinner fa-spin text-sm"></i>
            <span>Sending to PC...</span>
          {:else if webText.trim() === lastSentText && consecutiveSendCount >= 3}
            <i class="fa-solid fa-ban text-sm"></i>
            <span>Already Sent (Max 3)</span>
          {:else if sentTextToast}
            <i class="fa-solid fa-circle-check text-emerald-300"></i>
            <span>Sent to PC!</span>
          {:else}
            <i class="fa-solid fa-paper-plane"></i>
            <span>Send to PC</span>
          {/if}
        </button>
      </div>
    </div>
  </div>
</div>

<!-- Image Modal Preview -->
{#if isImageModalOpen && screenshot}
  <button
    type="button"
    class="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 cursor-pointer border-none w-full h-full text-left"
    on:click={() => (isImageModalOpen = false)}
  >
    <div class="relative max-w-5xl max-h-[90vh] flex flex-col items-center">
      <img
        src={screenshot}
        alt="Full Screen Preview"
        class="max-w-full max-h-[85vh] object-contain rounded-xl shadow-2xl border border-white/10"
      />
      <div class="mt-3 text-white text-xs font-semibold bg-black/60 px-3 py-1 rounded-full border border-white/20">
        Click anywhere to close
      </div>
    </div>
  </button>
{/if}
