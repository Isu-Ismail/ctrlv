<script lang="ts">
  import CodeBox from '../components/CodeBox.svelte';

  type OS = 'windows' | 'linux' | 'mac';
  type Arch = 'amd64' | 'arm64';

  let activeOS: OS = 'windows';
  let activeArch: Arch = 'amd64';

  // Windows commands & links
  $: winPsCmd = `Invoke-WebRequest -Uri https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_windows_${activeArch}.zip -OutFile ctrlv.zip; Expand-Archive ctrlv.zip -DestinationPath C:\\ctrlv`;
  $: winZipLink = `https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_windows_${activeArch}.zip`;
  const winPathCmd = `[System.Environment]::SetEnvironmentVariable("PATH", [System.Environment]::GetEnvironmentVariable("PATH", "User") + ";C:\\ctrlv", "User")`;

  // Linux commands & links
  $: linuxAptCmd = `wget https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_linux_${activeArch}.deb && sudo apt install ./ctrlv-cli_linux_${activeArch}.deb`;
  $: linuxDebLink = `https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_linux_${activeArch}.deb`;
  $: linuxRpmLink = `https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_linux_${activeArch}.rpm`;
  const linuxPathCmd = `echo 'export PATH="$PATH:/path/to/ctrlv-folder"' >> ~/.bashrc`;

  // macOS commands & links
  $: macArchTarget = activeArch === 'amd64' ? 'amd64' : 'arm64';
  $: macCurlCmd = `curl -sSfL https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_darwin_${macArchTarget}.tar.gz | tar -xz && sudo mv ctrlv /usr/local/bin/`;
  $: macTarLink = `https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_darwin_${macArchTarget}.tar.gz`;
  const macPathCmd = `echo 'export PATH="$PATH:/path/to/ctrlv-folder"' >> ~/.zshrc`;
</script>

<div class="max-w-[1400px] mx-auto p-4 sm:p-6 space-y-6">
  <div class="glass-panel p-6 sm:p-8 space-y-6">
    <div class="flex items-center justify-between flex-wrap gap-4 pb-4 border-b border-[var(--card-border)]">
      <!-- OS Navigation Tabs -->
      <div class="flex items-center gap-1.5 bg-[var(--bg-tertiary)] p-1.5 rounded-xl border border-[var(--card-border)]">
        <button
          on:click={() => (activeOS = 'windows')}
          class={`px-4 sm:px-5 py-2.5 rounded-lg text-xs sm:text-sm font-bold flex items-center gap-2 transition-all cursor-pointer ${
            activeOS === 'windows'
              ? 'bg-[var(--accent-primary)] text-white shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
          }`}
        >
          <i class="fa-brands fa-windows text-sky-400"></i>
          <span>Windows</span>
        </button>
        <button
          on:click={() => (activeOS = 'linux')}
          class={`px-4 sm:px-5 py-2.5 rounded-lg text-xs sm:text-sm font-bold flex items-center gap-2 transition-all cursor-pointer ${
            activeOS === 'linux'
              ? 'bg-[var(--accent-primary)] text-white shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
          }`}
        >
          <i class="fa-brands fa-linux text-amber-400"></i>
          <span>Linux</span>
        </button>
        <button
          on:click={() => (activeOS = 'mac')}
          class={`px-4 sm:px-5 py-2.5 rounded-lg text-xs sm:text-sm font-bold flex items-center gap-2 transition-all cursor-pointer ${
            activeOS === 'mac'
              ? 'bg-[var(--accent-primary)] text-white shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
          }`}
        >
          <i class="fa-brands fa-apple text-slate-300"></i>
          <span>macOS</span>
        </button>
      </div>

      <!-- Architecture Switcher -->
      <div class="flex items-center gap-2 text-xs sm:text-sm">
        <span class="font-bold text-[var(--text-muted)]">Arch:</span>
        <div class="flex items-center gap-1 bg-[var(--bg-tertiary)] p-1 rounded-xl border border-[var(--card-border)]">
          <button
            on:click={() => (activeArch = 'amd64')}
            class={`px-3 py-1.5 rounded-lg font-bold transition-all cursor-pointer ${
              activeArch === 'amd64'
                ? 'bg-[var(--bg-input)] text-[var(--accent-cyan)] shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            x86_64 / AMD64
          </button>
          <button
            on:click={() => (activeArch = 'arm64')}
            class={`px-3 py-1.5 rounded-lg font-bold transition-all cursor-pointer ${
              activeArch === 'arm64'
                ? 'bg-[var(--bg-input)] text-[var(--accent-cyan)] shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'
            }`}
          >
            ARM64
          </button>
        </div>
      </div>
    </div>

    <!-- Windows Section -->
    {#if activeOS === 'windows'}
      <div class="space-y-5 animate-fade-in">
        <div class="space-y-2">
          <h4 class="text-xs sm:text-sm font-extrabold text-[var(--text-main)]">Quick Install via PowerShell:</h4>
          <CodeBox promptSymbol="PS>" codeText={winPsCmd} />
        </div>

        <div class="pt-2">
          <a
            href={winZipLink}
            target="_blank"
            class="inline-flex items-center gap-2 px-5 py-3 rounded-xl bg-[var(--accent-primary)] hover:bg-[var(--accent-hover)] text-white text-xs sm:text-sm font-bold shadow-md shadow-indigo-500/25 transition-all text-decoration-none"
          >
            <i class="fa-solid fa-file-zipper"></i>
            <span>Download Portable ZIP ({activeArch.toUpperCase()})</span>
          </a>
        </div>

        <div class="space-y-2 pt-2 border-t border-[var(--card-border)]">
          <h4 class="text-xs sm:text-sm font-extrabold text-[var(--text-main)]">Add C:\ctrlv to PATH Environment Variable:</h4>
          <CodeBox promptSymbol="PS>" codeText={winPathCmd} />
        </div>
      </div>
    {/if}

    <!-- Linux Section -->
    {#if activeOS === 'linux'}
      <div class="space-y-5 animate-fade-in">
        <div class="space-y-2">
          <h4 class="text-xs sm:text-sm font-extrabold text-[var(--text-main)]">Debian / Ubuntu One-Line Install:</h4>
          <CodeBox promptSymbol="$" codeText={linuxAptCmd} />
        </div>

        <div class="flex items-center gap-3 flex-wrap pt-2">
          <a
            href={linuxDebLink}
            target="_blank"
            class="inline-flex items-center gap-2 px-5 py-3 rounded-xl bg-[var(--accent-primary)] hover:bg-[var(--accent-hover)] text-white text-xs sm:text-sm font-bold shadow-md shadow-indigo-500/25 transition-all text-decoration-none"
          >
            <i class="fa-solid fa-box-archive"></i>
            <span>Download .DEB ({activeArch.toUpperCase()})</span>
          </a>
          <a
            href={linuxRpmLink}
            target="_blank"
            class="inline-flex items-center gap-2 px-5 py-3 rounded-xl bg-[var(--bg-tertiary)] hover:bg-[var(--card-hover)] text-[var(--text-main)] border border-[var(--card-border)] text-xs sm:text-sm font-bold transition-all text-decoration-none"
          >
            <i class="fa-solid fa-box-open"></i>
            <span>Download .RPM ({activeArch.toUpperCase()})</span>
          </a>
        </div>

        <div class="space-y-2 pt-2 border-t border-[var(--card-border)]">
          <h4 class="text-xs sm:text-sm font-extrabold text-[var(--text-main)]">Add to PATH (~/.bashrc):</h4>
          <CodeBox promptSymbol="$" codeText={linuxPathCmd} />
        </div>
      </div>
    {/if}

    <!-- macOS Section -->
    {#if activeOS === 'mac'}
      <div class="space-y-5 animate-fade-in">
        <div class="space-y-2">
          <h4 class="text-xs sm:text-sm font-extrabold text-[var(--text-main)]">One-Line Terminal Install:</h4>
          <CodeBox promptSymbol="$" codeText={macCurlCmd} />
        </div>

        <div class="pt-2">
          <a
            href={macTarLink}
            target="_blank"
            class="inline-flex items-center gap-2 px-5 py-3 rounded-xl bg-[var(--accent-primary)] hover:bg-[var(--accent-hover)] text-white text-xs sm:text-sm font-bold shadow-md shadow-indigo-500/25 transition-all text-decoration-none"
          >
            <i class="fa-solid fa-file-arrow-down"></i>
            <span>Download .TAR.GZ ({macArchTarget.toUpperCase()})</span>
          </a>
        </div>

        <div class="space-y-2 pt-2 border-t border-[var(--card-border)]">
          <h4 class="text-xs sm:text-sm font-extrabold text-[var(--text-main)]">Add to PATH (~/.zshrc):</h4>
          <CodeBox promptSymbol="$" codeText={macPathCmd} />
        </div>
      </div>
    {/if}
  </div>
</div>
