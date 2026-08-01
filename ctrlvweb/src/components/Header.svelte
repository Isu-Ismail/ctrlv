<script lang="ts">
  import { themeStore, toggleTheme } from '../lib/stores/themeStore';
  import { wsStore, setRoomId } from '../lib/stores/wsStore';
  import { activeTabStore, setActiveTab } from '../lib/stores/viewStore';
  import { connectWebSocket, disconnectWebSocket } from '../lib/services/wsService';

  let roomIdInput = $wsStore.roomId;
  let copiedRoomToast = false;
  let copiedCmdToast = false;

  $: isConnected = $wsStore.isConnected;
  $: isConnecting = $wsStore.isConnecting;
  $: browserCount = $wsStore.browserCount;
  $: pcCount = $wsStore.pcCount;

  function handleRoomInputChange() {
    if (roomIdInput && roomIdInput.trim()) {
      setRoomId(roomIdInput.trim());
      if (isConnected) {
        disconnectWebSocket();
        setTimeout(() => connectWebSocket(), 150);
      }
    }
  }

  function handleConnectToggle() {
    if (isConnected) {
      disconnectWebSocket();
    } else {
      setRoomId(roomIdInput);
      connectWebSocket();
    }
  }

  function generateRandomRoom() {
    const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
    let rand = '';
    for (let i = 0; i < 6; i++) {
      rand += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    const newRoom = `ctrlv-${rand}`;
    roomIdInput = newRoom;
    setRoomId(newRoom);
    if (isConnected) {
      disconnectWebSocket();
      setTimeout(() => connectWebSocket(), 150);
    }
    copyRoomId();
  }

  async function copyRoomId() {
    try {
      await navigator.clipboard.writeText(roomIdInput);
      copiedRoomToast = true;
      setTimeout(() => (copiedRoomToast = false), 1800);
    } catch (e) {}
  }

  async function copyCliCmd() {
    try {
      const cmd = `ctrlv -r ${roomIdInput} -s`;
      await navigator.clipboard.writeText(cmd);
      copiedCmdToast = true;
      setTimeout(() => (copiedCmdToast = false), 1800);
    } catch (e) {}
  }
</script>

<header class="sticky top-0 z-40 bg-[var(--bg-secondary)]/95 backdrop-blur-lg border-b border-[var(--card-border)] px-3 sm:px-6 py-2.5 sm:py-3 shadow-md">
  <div class="max-w-[1400px] mx-auto flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3 sm:gap-4">

    <!-- Top Row on Mobile / Left Group on Desktop -->
    <div class="flex items-center justify-between md:justify-start gap-2 sm:gap-4 w-full md:w-auto">
      <div class="flex items-center gap-2.5">
        <button on:click={() => setActiveTab('dashboard')} class="flex items-center gap-2 sm:gap-3 group border-none bg-transparent cursor-pointer p-0 text-left">
          <!-- Vibrant Indigo Blue Brand Icon with White Logo SVG -->
          <div class="w-9 h-9 sm:w-10 sm:h-10 rounded-xl bg-gradient-to-br from-indigo-600 to-indigo-700 border border-indigo-500/40 flex items-center justify-center text-white group-hover:scale-105 transition-transform shadow-md">
            <svg class="w-5 h-5 sm:w-6 sm:h-6" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M8 2C7.44772 2 7 2.44772 7 3C7 3.55228 7.44772 4 8 4H10C10.5523 4 11 3.55228 11 3C11 2.44772 10.5523 2 10 2H8Z" fill="#ffffff"/>
              <path d="M3 5C3 3.89543 3.89543 3 5 3C5 4.65685 6.34315 6 8 6H10C11.6569 6 13 4.65685 13 3C14.1046 3 15 3.89543 15 5V11H10.4142L11.7071 9.70711C12.0976 9.31658 12.0976 8.68342 11.7071 8.29289C11.3166 7.90237 10.6834 7.90237 10.2929 8.29289L7.29289 11.2929C6.90237 11.6834 6.90237 12.3166 7.29289 12.7071L10.2929 15.7071C10.6834 16.0976 11.3166 16.0976 11.7071 15.7071C12.0976 15.3166 12.0976 14.6834 11.7071 14.2929L10.4142 13H15V16C15 17.1046 14.1046 18 13 18H5C3.89543 18 3 17.1046 3 16V5Z" fill="#ffffff"/>
              <path d="M15 11H17C17.5523 11 18 11.4477 18 12C18 12.5523 17.5523 13 17 13H15V11Z" fill="#ffffff"/>
            </svg>
          </div>
          <div class="flex flex-col leading-tight">
            <span class="font-black text-base sm:text-lg tracking-tight text-[var(--text-main)]">ctrlv</span>
            <span class="text-[10px] sm:text-xs font-extrabold text-[var(--text-muted)] opacity-85 hidden sm:inline">v1.0.0</span>
          </div>
        </button>

        <!-- Connection Status Badge -->
        <div
          class={`inline-flex items-center gap-1.5 sm:gap-2 px-2.5 sm:px-3.5 py-1 sm:py-1.5 rounded-full text-xs font-bold border transition-all ${
            isConnected
              ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
              : isConnecting
              ? 'bg-amber-500/10 text-amber-400 border-amber-500/30 animate-pulse'
              : 'bg-rose-500/10 text-rose-400 border-rose-500/30'
          }`}
        >
          {#if isConnecting}
            <i class="fa-solid fa-spinner fa-spin text-amber-400"></i>
            <span class="hidden sm:inline">Connecting...</span>
          {:else if isConnected}
            <i class="fa-solid fa-signal text-emerald-400"></i>
            <span class="hidden sm:inline">Live Sync</span>
          {:else}
            <i class="fa-solid fa-plug text-rose-400"></i>
            <span class="hidden sm:inline">Disconnected</span>
          {/if}
        </div>

        <!-- Client Counts Badge -->
        {#if isConnected}
          <div class="inline-flex items-center gap-2 sm:gap-3.5 px-2.5 sm:px-3.5 py-1 sm:py-1.5 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--card-border)] text-xs text-[var(--text-muted)]">
            <span title="Web Browser Clients Connected" class="inline-flex items-center gap-1">
              <i class="fa-solid fa-globe text-cyan-400"></i>
              <span class="font-extrabold text-[var(--text-main)]">{browserCount}</span>
            </span>
            <span title="PC CLI Clients Connected" class="inline-flex items-center gap-1">
              <i class="fa-solid fa-laptop text-[var(--accent-primary)]"></i>
              <span class="font-extrabold text-[var(--text-main)]">{pcCount}</span>
            </span>
          </div>
        {/if}
      </div>

      <!-- Theme Switcher Button (Mobile view) -->
      <button
        on:click={toggleTheme}
        class="w-10 h-10 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--card-border)] text-[var(--text-muted)] hover:text-[var(--accent-primary)] hover:border-[var(--accent-primary)]/40 hover:bg-[var(--accent-primary)]/10 flex items-center justify-center transition-colors cursor-pointer shrink-0 md:hidden"
        title="Toggle Dark/Light Theme"
      >
        {#if $themeStore === 'dark'}
          <i class="fa-solid fa-moon text-indigo-400 text-xs sm:text-sm"></i>
        {:else}
          <i class="fa-solid fa-sun text-amber-400 text-xs sm:text-sm"></i>
        {/if}
      </button>
    </div>

    <!-- Room ID Selector Row -->
    <div class="flex items-center justify-between gap-1 sm:gap-1.5 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl p-1.5 text-xs w-full md:w-auto h-11">
      <div class="flex items-center gap-1.5 min-w-0 flex-1">
        <i class="fa-solid fa-door-open text-[var(--text-muted)] pl-1.5 shrink-0"></i>
        <input
          type="text"
          bind:value={roomIdInput}
          on:change={handleRoomInputChange}
          placeholder="Room ID"
          class="bg-transparent border-none outline-none text-[var(--text-main)] font-mono text-xs font-bold w-full min-w-0 px-1"
        />
      </div>
      <div class="flex items-center gap-1 shrink-0">
        <button
          on:click={copyRoomId}
          class="p-1.5 px-2 rounded-lg text-[var(--text-muted)] hover:text-[var(--accent-primary)] hover:bg-[var(--accent-primary)]/15 transition-colors cursor-pointer"
          title="Copy Room ID"
        >
          {#if copiedRoomToast}
            <i class="fa-solid fa-check text-emerald-400"></i>
          {:else}
            <i class="fa-regular fa-copy"></i>
          {/if}
        </button>
        <button
          on:click={generateRandomRoom}
          class="p-1.5 px-2 rounded-lg text-[var(--text-muted)] hover:text-[var(--accent-primary)] hover:bg-[var(--accent-primary)]/15 transition-colors cursor-pointer"
          title="Generate Random Room ID"
        >
          <i class="fa-solid fa-dice"></i>
        </button>
        <button
          on:click={copyCliCmd}
          class="p-1.5 px-2 rounded-lg text-[var(--text-muted)] hover:text-[var(--accent-primary)] hover:bg-[var(--accent-primary)]/15 transition-colors cursor-pointer"
          title="Copy CLI Command (ctrlv -r room -s)"
        >
          {#if copiedCmdToast}
            <i class="fa-solid fa-check text-emerald-400"></i>
          {:else}
            <i class="fa-solid fa-terminal"></i>
          {/if}
        </button>
      </div>
    </div>

    <!-- Navigation Tabs on LEFT, Compact Connect Button on RIGHT (All exactly h-11 high) -->
    <div class="flex items-center gap-2 w-full md:w-auto">
      <!-- Router Nav Tabs Container (h-11) -->
      <nav class="flex items-center gap-1 bg-[var(--bg-tertiary)] p-1 rounded-xl border border-[var(--card-border)] flex-1 md:flex-none h-11">
        <!-- 1. Sync -->
        <button
          on:click={() => setActiveTab('dashboard')}
          class={`flex-1 md:flex-none h-9 px-3.5 rounded-lg text-xs font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 ${
            $activeTabStore === 'dashboard'
              ? 'bg-[var(--accent-primary)] text-white shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--text-main)] hover:bg-[var(--card-hover)]'
          }`}
          title="Sync View"
        >
          <i class="fa-solid fa-rotate text-sm"></i>
          <span class="hidden sm:inline">Sync</span>
        </button>

        <!-- 2. History -->
        <button
          on:click={() => setActiveTab('history')}
          class={`flex-1 md:flex-none h-9 px-3.5 rounded-lg text-xs font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 ${
            $activeTabStore === 'history'
              ? 'bg-[var(--accent-primary)] text-white shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--text-main)] hover:bg-[var(--card-hover)]'
          }`}
          title="Room History"
        >
          <i class="fa-solid fa-clock-rotate-left text-sm"></i>
          <span class="hidden sm:inline">History</span>
        </button>

        <!-- 3. Config -->
        <button
          on:click={() => setActiveTab('config')}
          class={`flex-1 md:flex-none h-9 px-3.5 rounded-lg text-xs font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 ${
            $activeTabStore === 'config'
              ? 'bg-[var(--accent-primary)] text-white shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--text-main)] hover:bg-[var(--card-hover)]'
          }`}
          title="AI Settings"
        >
          <i class="fa-solid fa-sliders text-sm"></i>
          <span class="hidden sm:inline">Config</span>
        </button>

        <!-- 4. CLI / Download -->
        <button
          on:click={() => setActiveTab('download')}
          class={`flex-1 md:flex-none h-9 px-3.5 rounded-lg text-xs font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 ${
            $activeTabStore === 'download'
              ? 'bg-[var(--accent-primary)] text-white shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--text-main)] hover:bg-[var(--card-hover)]'
          }`}
          title="CLI Downloads"
        >
          <i class="fa-solid fa-download text-sm"></i>
          <span class="hidden sm:inline">CLI</span>
        </button>
      </nav>

      <!-- Connect Button (Exact same height h-11) -->
      <button
        on:click={handleConnectToggle}
        disabled={isConnecting}
        class={`shrink-0 h-11 px-5 rounded-xl font-extrabold text-sm flex items-center justify-center gap-2 transition-all cursor-pointer shadow-md disabled:opacity-60 disabled:cursor-not-allowed ${
          isConnected
            ? 'bg-rose-600 hover:bg-rose-700 text-white shadow-rose-600/20'
            : isConnecting
            ? 'bg-amber-600 text-white shadow-amber-600/20'
            : 'bg-[var(--accent-primary)] hover:bg-[var(--accent-hover)] text-white shadow-indigo-500/25'
        }`}
      >
        {#if isConnecting}
          <i class="fa-solid fa-spinner fa-spin text-sm"></i>
          <span>Connecting...</span>
        {:else if isConnected}
          <i class="fa-solid fa-plug-circle-xmark text-sm"></i>
          <span>Disconnect</span>
        {:else}
          <i class="fa-solid fa-plug text-sm"></i>
          <span>Connect</span>
        {/if}
      </button>

      <!-- Theme Switcher Button (Desktop, exact same height h-11 and width w-11) -->
      <button
        on:click={toggleTheme}
        class="w-11 h-11 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--card-border)] text-[var(--text-muted)] hover:text-[var(--accent-primary)] hover:border-[var(--accent-primary)]/40 hover:bg-[var(--accent-primary)]/10 hidden md:flex items-center justify-center transition-colors cursor-pointer shrink-0"
        title="Toggle Dark/Light Theme"
      >
        {#if $themeStore === 'dark'}
          <i class="fa-solid fa-moon text-indigo-400 text-sm"></i>
        {:else}
          <i class="fa-solid fa-sun text-amber-400 text-sm"></i>
        {/if}
      </button>
    </div>

  </div>
</header>
