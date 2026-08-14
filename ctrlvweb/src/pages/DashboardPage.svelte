<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { setActiveTab } from '../lib/stores/viewStore';
  import {
    wsStore,
    setAutoDownload,
    setAutoSolve,
    setAutoPush,
    setWebText,
    setCachedScreenshot
  } from '../lib/stores/wsStore';
  import { aiConfigStore, aiSolverStatusStore } from '../lib/stores/aiStore';
  import { historyStore } from '../lib/stores/historyStore';
  import { sendTextToPC, triggerImageDownload } from '../lib/services/wsService';
  import { solveImageWithAI } from '../lib/services/aiService';

  $: screenshot = $wsStore.cachedScreenshot;
  $: pcText = $wsStore.cachedPCText;
  $: webText = $wsStore.webText;
  $: roomId = $wsStore.roomId;
  $: autoSolve = $wsStore.autoSolve;
  $: autoPush = $wsStore.autoPush;

  $: aiConfig = $aiConfigStore;
  $: aiStatus = $aiSolverStatusStore;
  $: historyItems = $historyStore;

  let activeTab: 'screenshot' | 'camera' | 'pctext' = 'screenshot';
  let selectedSnippetTab: 'main' | number = 'main';
  let isImageModalOpen = false;
  let modalImageSrc = '';
  let copiedTextToast = false;
  let sentTextToast = false;
  let capturedToast = false;

  // Tab-specific Auto Download / Save toggles
  let autoDownloadScreenshot = typeof window !== 'undefined' ? localStorage.getItem('ctrlv_auto_download_screenshot') !== 'false' : false;
  let autoDownloadCamera = typeof window !== 'undefined' ? localStorage.getItem('ctrlv_auto_download_camera') === 'true' : false;

  function toggleAutoDownloadScreenshot() {
    autoDownloadScreenshot = !autoDownloadScreenshot;
    setAutoDownload(autoDownloadScreenshot);
    if (typeof window !== 'undefined') {
      localStorage.setItem('ctrlv_auto_download_screenshot', autoDownloadScreenshot ? 'true' : 'false');
    }
  }

  function toggleAutoDownloadCamera() {
    autoDownloadCamera = !autoDownloadCamera;
    if (typeof window !== 'undefined') {
      localStorage.setItem('ctrlv_auto_download_camera', autoDownloadCamera ? 'true' : 'false');
    }
  }

  // Send limit and locking state
  let lastSentText = '';
  let consecutiveSendCount = 0;
  let isSendingToPC = false;

  // Camera Mode state
  let videoElement: HTMLVideoElement;
  let canvasElement: HTMLCanvasElement;
  let mediaStream: MediaStream | null = null;
  let facingMode: 'environment' | 'user' = 'environment';
  let isCameraActive = false;
  let cameraError = '';

  async function startCamera() {
    stopCamera();
    cameraError = '';
    try {
      mediaStream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: facingMode, width: { ideal: 1920 }, height: { ideal: 1080 } }
      });
      if (videoElement) {
        videoElement.srcObject = mediaStream;
        await videoElement.play();
        isCameraActive = true;
      }
    } catch (e: any) {
      console.error('Camera access error:', e);
      cameraError = e?.message || 'Could not access camera. Please allow camera permissions.';
      isCameraActive = false;
    }
  }

  function stopCamera() {
    if (mediaStream) {
      mediaStream.getTracks().forEach((track) => track.stop());
      mediaStream = null;
    }
    if (videoElement) {
      videoElement.srcObject = null;
    }
    isCameraActive = false;
  }

  function toggleFacingMode() {
    facingMode = facingMode === 'environment' ? 'user' : 'environment';
    if (activeTab === 'camera') {
      startCamera();
    }
  }

  function switchTab(tab: 'screenshot' | 'camera' | 'pctext') {
    activeTab = tab;
    if (tab === 'camera') {
      setTimeout(() => startCamera(), 100);
    } else {
      stopCamera();
    }
  }

  async function captureFrame() {
    if (!videoElement || !isCameraActive) {
      if (cameraError || !isCameraActive) {
        await startCamera();
      }
      return;
    }
    try {
      const canvas = canvasElement || document.createElement('canvas');
      canvas.width = videoElement.videoWidth || 1280;
      canvas.height = videoElement.videoHeight || 720;
      const ctx = canvas.getContext('2d');
      if (ctx) {
        ctx.drawImage(videoElement, 0, 0, canvas.width, canvas.height);
        const b64Data = canvas.toDataURL('image/jpeg', 0.9);
        setCachedScreenshot(b64Data, roomId);
        
        // Show feedback toast on Capture button
        capturedToast = true;
        setTimeout(() => (capturedToast = false), 1500);

        // Auto Download / Save captured camera image if Save ON for Camera
        if (autoDownloadCamera) {
          triggerImageDownload(b64Data, `ctrlv_camera_${roomId}_${Date.now()}.jpg`);
        }

        // Obey Auto-Solve setting: if autoSolve is ON, trigger AI solving automatically
        if (autoSolve) {
          await solveImageWithAI(b64Data);
        }
      }
    } catch (e) {
      console.error('Failed to capture camera frame:', e);
    }
  }

  onDestroy(() => {
    stopCamera();
  });

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

  async function copyText(textToCopy: string) {
    if (!textToCopy) return;
    try {
      await navigator.clipboard.writeText(textToCopy);
      copiedTextToast = true;
      setTimeout(() => (copiedTextToast = false), 1800);
    } catch (e) {}
  }

  function loadHistoryToMain(itemText: string) {
    if (!itemText) return;
    setWebText(itemText, roomId);
    selectedSnippetTab = 'main';
  }

  async function handleSolveClick() {
    if (activeTab === 'screenshot' && screenshot) {
      await solveImageWithAI(screenshot);
    } else if (activeTab === 'pctext' && pcText) {
      await solveImageWithAI(null, pcText);
    } else if (activeTab === 'camera') {
      await captureFrame();
    }
  }

  function handleDownloadClick() {
    if (screenshot) {
      triggerImageDownload(screenshot, `ctrlv_screenshot_${roomId}_${Date.now()}.jpg`);
    }
  }

  function openImageModal(imgSrc: string) {
    modalImageSrc = imgSrc;
    isImageModalOpen = true;
  }
</script>

<canvas bind:this={canvasElement} class="hidden"></canvas>

<!-- PC View: Fits perfectly in viewport (md:h-[calc(100vh-156px)]) with equal top & bottom padding. Mobile View: Normal scrollable -->
<div class="max-w-[1400px] w-full mx-auto p-4 sm:p-5 md:h-[calc(100vh-156px)] md:overflow-hidden flex flex-col justify-between my-auto">
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 flex-1 min-h-0 h-full">

    <!-- LEFT COLUMN: Snippet / AI Output & History Tabs -->
    <div class="glass-panel p-4 sm:p-5 flex flex-col gap-3.5 h-full min-h-0">
      <!-- Header Row 1: Title & Toggles -->
      <div class="flex items-center justify-between flex-wrap gap-2 pb-2 border-b border-[var(--card-border)]/60 shrink-0">
        <div class="flex items-center gap-2 font-bold text-sm sm:text-base text-[var(--text-main)]">
          <i class="fa-solid fa-code text-[var(--accent-primary)]"></i>
          <span>Snippet</span>
        </div>

        <div class="flex items-center gap-2 flex-wrap">
          <!-- Push Toggle Button -->
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
            <span class="hidden sm:inline">Push: {autoPush ? 'ON' : 'OFF'}</span>
          </button>

          <!-- AI Solver Status Pill (Icon only on mobile, text on sm+) -->
          <div
            title={aiStatus.message}
            class={`px-2.5 sm:px-3 py-1.5 sm:py-1 rounded-full text-xs sm:text-sm font-bold border flex items-center gap-1.5 ${
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
            <span class="hidden sm:inline">{aiStatus.message}</span>
          </div>
        </div>
      </div>

      <!-- Header Row 2: Sub-Tabs (Pinned 'Main' + Scrollable 'H1'..'H10' with hidden scrollbar) -->
      <div class="flex items-center gap-2 pb-2 border-b border-[var(--card-border)] shrink-0 min-w-0">
        <!-- Pinned Main Button -->
        <button
          on:click={() => (selectedSnippetTab = 'main')}
          class={`px-3.5 py-1.5 rounded-lg text-xs font-black transition-all cursor-pointer shrink-0 border flex items-center gap-1.5 ${
            selectedSnippetTab === 'main'
              ? 'bg-[var(--accent-primary)] text-white border-[var(--accent-primary)] shadow-md'
              : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--card-border)] hover:text-[var(--text-main)]'
          }`}
        >
          <i class="fa-solid fa-house text-[10px]"></i>
          <span>Main</span>
        </button>

        <div class="h-4 w-[1px] bg-[var(--card-border)] shrink-0"></div>

        <!-- Scrollable History Tabs with Hidden Scrollbar -->
        <div class="flex items-center gap-1.5 overflow-x-auto flex-1 min-w-0 py-0.5" style="scrollbar-width: none; -ms-overflow-style: none;">
          {#each historyItems.slice(0, 10) as item, idx}
            <button
              on:click={() => (selectedSnippetTab = idx)}
              class={`px-3 py-1.5 rounded-lg text-xs font-bold transition-all cursor-pointer whitespace-nowrap flex items-center gap-1 border shrink-0 ${
                selectedSnippetTab === idx
                  ? 'bg-[var(--accent-primary)] text-white border-[var(--accent-primary)] shadow-sm'
                  : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--card-border)] hover:text-[var(--text-main)]'
              }`}
              title={`${item.time} - ${item.text.substring(0, 30)}...`}
            >
              {#if item.image}
                <i class="fa-solid fa-camera text-[10px] text-amber-300"></i>
              {/if}
              <span>H{idx + 1}</span>
            </button>
          {/each}
          {#if historyItems.length === 0}
            <span class="text-[11px] text-[var(--text-dim)] italic px-1">No history yet</span>
          {/if}
        </div>
      </div>

      <!-- Live Web Text Area (Main) vs Read-only History Area (H1..H10) -->
      {#if selectedSnippetTab === 'main'}
        <div class="relative flex-1 min-h-[290px] bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl flex flex-col p-3.5">
          <textarea
            value={webText}
            on:input={handleWebTextChange}
            placeholder="Type or paste text here to automatically sync with your PC clipboard in real-time..."
            class="w-full flex-1 bg-transparent border-none outline-none font-mono text-xs sm:text-sm text-[var(--text-main)] resize-none custom-scrollbar p-1 leading-relaxed"
          ></textarea>
        </div>
      {:else}
        <!-- History View (H1..H10) -->
        {@const historyItem = historyItems[selectedSnippetTab]}
        {#if historyItem}
          <div class="relative flex-1 min-h-[290px] bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl flex flex-col p-3.5 gap-2 overflow-y-auto custom-scrollbar">
            {#if historyItem.image}
              <div class="flex items-center gap-3 p-2 bg-[var(--bg-tertiary)] border border-[var(--card-border)] rounded-lg shrink-0">
                <button
                  type="button"
                  on:click={() => openImageModal(historyItem.image!)}
                  class="border-none bg-transparent p-0 cursor-zoom-in shrink-0"
                  title="Click to expand full image"
                >
                  <img
                    src={historyItem.image}
                    alt={`Screenshot for H${selectedSnippetTab + 1}`}
                    class="w-16 h-16 object-contain rounded border border-[var(--card-border)] hover:scale-105 transition-transform"
                  />
                </button>
                <div class="flex flex-col text-xs text-[var(--text-muted)] leading-tight">
                  <span class="font-bold text-[var(--text-main)] flex items-center gap-1">
                    <i class="fa-solid fa-camera text-amber-400"></i> Captured Image Attached
                  </span>
                  <span>Time: {historyItem.time} ({historyItem.date})</span>
                </div>
              </div>
            {/if}

            <textarea
              value={historyItem.text}
              readonly
              class="w-full flex-1 bg-transparent border-none outline-none font-mono text-xs sm:text-sm text-[var(--text-main)] resize-none custom-scrollbar p-1 leading-relaxed"
            ></textarea>
          </div>
        {:else}
          <div class="relative flex-1 min-h-[260px] bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl flex items-center justify-center p-4 text-center text-[var(--text-muted)] text-xs font-semibold">
            No history entry stored for H{selectedSnippetTab + 1}
          </div>
        {/if}
      {/if}

      <!-- Action Footer (Send to PC & optional Capture Button in Camera mode) -->
      <div class="flex items-center justify-between gap-3 pt-2 border-t border-[var(--card-border)] flex-wrap shrink-0">
        <span class="text-xs text-[var(--text-muted)] flex items-center gap-1.5">
          <i class="fa-solid fa-info-circle text-[var(--accent-cyan)]"></i>
          {#if selectedSnippetTab !== 'main'}
            <span>Viewing saved history H{selectedSnippetTab + 1}</span>
          {:else if activeTab === 'camera'}
            <span>Tap Capture to snap photo frame</span>
          {:else}
            <span>Text auto-syncs to PC clipboard upon sending</span>
          {/if}
        </span>

        <div class="flex items-center gap-2.5 w-full sm:w-auto">
          {#if selectedSnippetTab !== 'main'}
            {@const historyItem = historyItems[selectedSnippetTab]}
            {#if historyItem}
              <button
                on:click={() => copyText(historyItem.text)}
                class="flex-1 sm:flex-none px-4 py-2.5 rounded-xl bg-[var(--bg-tertiary)] hover:bg-[var(--card-hover)] text-xs sm:text-sm font-semibold text-[var(--text-main)] border border-[var(--card-border)] flex items-center justify-center gap-1.5 transition-all cursor-pointer"
              >
                {#if copiedTextToast}
                  <i class="fa-solid fa-check text-emerald-400"></i>
                  <span>Copied!</span>
                {:else}
                  <i class="fa-regular fa-copy"></i>
                  <span>Copy</span>
                {/if}
              </button>

              <button
                on:click={() => loadHistoryToMain(historyItem.text)}
                class="flex-1 sm:flex-none px-5 py-2.5 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white font-bold text-xs sm:text-sm flex items-center justify-center gap-1.5 shadow-lg transition-all cursor-pointer"
              >
                <i class="fa-solid fa-arrow-turn-up"></i>
                <span>Load to Main</span>
              </button>
            {/if}
          {:else}
            {#if activeTab === 'camera'}
              <button
                on:click={captureFrame}
                class="flex-1 sm:flex-none px-5 py-2.5 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-black text-xs sm:text-sm flex items-center justify-center gap-2 shadow-lg shadow-emerald-500/25 transition-all cursor-pointer"
              >
                {#if capturedToast}
                  <i class="fa-solid fa-circle-check text-emerald-300 text-sm"></i>
                  <span>Captured!</span>
                {:else}
                  <i class="fa-solid fa-camera text-sm"></i>
                  <span>Capture</span>
                {/if}
              </button>
            {/if}

            <button
              on:click={handleSendToPC}
              disabled={!webText || !webText.trim() || isSendingToPC || (webText.trim() === lastSentText && consecutiveSendCount >= 3)}
              class={`flex-1 sm:flex-none px-6 py-2.5 rounded-xl text-white font-extrabold text-xs sm:text-sm flex items-center justify-center gap-2 shadow-lg transition-all cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed ${
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
          {/if}
        </div>
      </div>
    </div>

    <!-- RIGHT COLUMN: Screenshot / Camera / PC Text Viewer -->
    <div class="glass-panel p-4 sm:p-5 flex flex-col gap-3.5 h-full min-h-0">
      <div class="flex items-center justify-between flex-wrap gap-2.5 pb-2.5 border-b border-[var(--card-border)] shrink-0">
        <!-- 3 Switcher Tabs in requested order: Screenshot | Camera | PC Text -->
        <div class="flex items-center gap-1 bg-[var(--bg-tertiary)] p-1 rounded-lg border border-[var(--card-border)] w-full sm:w-auto">
          <button
            on:click={() => switchTab('screenshot')}
            class={`flex-1 sm:flex-none px-3.5 py-1.5 rounded-md text-xs sm:text-sm font-bold flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
              activeTab === 'screenshot'
                ? 'bg-[var(--accent-primary)] text-white shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            <i class="fa-solid fa-desktop"></i>
            <span>Screenshot</span>
          </button>

          <button
            on:click={() => switchTab('camera')}
            class={`flex-1 sm:flex-none px-3.5 py-1.5 rounded-md text-xs sm:text-sm font-bold flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
              activeTab === 'camera'
                ? 'bg-[var(--accent-primary)] text-white shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            <i class="fa-solid fa-camera"></i>
            <span>Camera</span>
          </button>

          <button
            on:click={() => switchTab('pctext')}
            class={`flex-1 sm:flex-none px-3.5 py-1.5 rounded-md text-xs sm:text-sm font-bold flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
              activeTab === 'pctext'
                ? 'bg-[var(--accent-primary)] text-white shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            <i class="fa-solid fa-keyboard"></i>
            <span>PC Text</span>
          </button>
        </div>

        <!-- Right Header Action Controls & Toggles (Per-Tab Independent Save Toggles) -->
        <div class="flex items-center gap-1.5 w-full sm:w-auto flex-wrap">
          {#if activeTab === 'camera'}
            <button
              on:click={toggleFacingMode}
              class="px-2.5 py-1.5 rounded-md text-xs font-bold text-[var(--accent-cyan)] bg-[var(--bg-tertiary)] hover:bg-[var(--card-hover)] border border-[var(--card-border)] flex items-center gap-1.5 transition-all cursor-pointer"
              title="Switch between front and back camera"
            >
              <i class="fa-solid fa-camera-rotate"></i>
              <span>Flip ({facingMode === 'environment' ? 'Back' : 'Front'})</span>
            </button>

            <button
              on:click={toggleAutoDownloadCamera}
              class={`flex-1 sm:flex-none justify-center px-2.5 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 border transition-all cursor-pointer ${
                autoDownloadCamera
                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                  : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--card-border)]'
              }`}
              title="Auto-save captured camera photos to browser downloads"
            >
              <i class="fa-solid fa-floppy-disk"></i>
              <span>Save: {autoDownloadCamera ? 'ON' : 'OFF'}</span>
            </button>
          {:else if activeTab === 'screenshot'}
            <button
              on:click={toggleAutoDownloadScreenshot}
              class={`flex-1 sm:flex-none justify-center px-2.5 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 border transition-all cursor-pointer ${
                autoDownloadScreenshot
                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                  : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--card-border)]'
              }`}
              title="Auto-save incoming screenshots to browser downloads"
            >
              <i class="fa-solid fa-floppy-disk"></i>
              <span>Save: {autoDownloadScreenshot ? 'ON' : 'OFF'}</span>
            </button>
          {/if}

          <!-- Auto-Solve Toggle (Available across all tabs) -->
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
              on:click={() => openImageModal(screenshot!)}
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
      {:else if activeTab === 'camera'}
        <!-- CAMERA LIVE VIEW -->
        <div class="relative flex-1 min-h-[260px] bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl flex items-center justify-center p-2 overflow-hidden">
          {#if cameraError}
            <div class="text-center space-y-3 py-8 px-4">
              <i class="fa-solid fa-video-slash text-5xl text-rose-400"></i>
              <div class="font-bold text-sm text-[var(--text-main)]">{cameraError}</div>
              <button
                on:click={startCamera}
                class="px-4 py-2 rounded-xl bg-[var(--accent-primary)] text-white text-xs font-bold shadow cursor-pointer"
              >
                <i class="fa-solid fa-rotate-right"></i> Retry Camera
              </button>
            </div>
          {:else}
            <!-- Live Video Stream -->
            <video
              bind:this={videoElement}
              autoplay
              playsinline
              muted
              class="max-h-full max-w-full object-contain rounded-lg shadow"
            ></video>
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
        {:else if activeTab === 'camera'}
          <button
            on:click={startCamera}
            class="flex-1 sm:flex-none px-4 py-2 rounded-xl bg-[var(--bg-tertiary)] hover:bg-[var(--card-hover)] text-xs sm:text-sm font-semibold text-[var(--text-main)] border border-[var(--card-border)] flex items-center justify-center gap-2 transition-all cursor-pointer"
          >
            <i class="fa-solid fa-rotate-right"></i>
            <span>Restart Cam</span>
          </button>
        {:else}
          <button
            on:click={() => copyText(pcText)}
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

  </div>
</div>

<!-- Image Modal Preview (Supports both live screenshot & history item images) -->
{#if isImageModalOpen && modalImageSrc}
  <button
    type="button"
    class="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 cursor-pointer border-none w-full h-full text-left"
    on:click={() => (isImageModalOpen = false)}
  >
    <div class="relative max-w-5xl max-h-[90vh] flex flex-col items-center">
      <img
        src={modalImageSrc}        alt="Full Screen Preview"
        class="max-w-full max-h-[85vh] object-contain rounded-xl shadow-2xl border border-white/10"
      />
      <div class="mt-3 text-white text-xs font-semibold bg-black/60 px-3 py-1 rounded-full border border-white/20">
        Click anywhere to close
      </div>
    </div>
  </button>
{/if}
