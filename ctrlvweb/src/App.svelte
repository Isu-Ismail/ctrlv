<script lang="ts">
  import Header from './components/Header.svelte';
  import Footer from './components/Footer.svelte';
  import ErrorModal from './components/ErrorModal.svelte';
  import DashboardPage from './pages/DashboardPage.svelte';
  import DownloadPage from './pages/DownloadPage.svelte';
  import ConfigPage from './pages/ConfigPage.svelte';
  import HistoryPage from './pages/HistoryPage.svelte';
  import { activeTabStore } from './lib/stores/viewStore';
</script>

<div class="min-h-screen flex flex-col bg-[var(--bg-main)] text-[var(--text-main)] antialiased selection:bg-[var(--accent-primary)] selection:text-white">
  <Header />

  <main class="flex-1 py-4">
    {#if $activeTabStore === 'dashboard'}
      <DashboardPage />
    {:else if $activeTabStore === 'download'}
      <DownloadPage />
    {:else if $activeTabStore === 'config'}
      <ConfigPage />
    {:else if $activeTabStore === 'history'}
      <HistoryPage />
    {/if}
  </main>

  <Footer />
  <ErrorModal />
</div>
