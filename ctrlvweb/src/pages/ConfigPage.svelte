<script lang="ts">
  import { aiConfigStore, resetAIConfig } from "../lib/stores/aiStore";
  import { setRelayUrl, wsStore } from "../lib/stores/wsStore";

  let provider = $aiConfigStore.provider;
  let model = $aiConfigStore.model;
  let apiKey = $aiConfigStore.apiKey;
  let maxTokens = $aiConfigStore.maxTokens;
  let customPrompt = $aiConfigStore.customPrompt;
  let codeOnly = $aiConfigStore.codeOnly;

  let currentRelayUrl = $wsStore.relayUrl;

  let showApiKey = false;
  let saveToast = false;
  let resetToast = false;

  function handleSave() {
    aiConfigStore.set({
      provider,
      model,
      apiKey: apiKey ? apiKey.trim() : "",
      maxTokens: Number(maxTokens) || 2048,
      customPrompt: customPrompt ? customPrompt.trim() : "",
      codeOnly,
    });

    if (currentRelayUrl && currentRelayUrl.trim()) {
      setRelayUrl(currentRelayUrl.trim());
    }

    saveToast = true;
    setTimeout(() => (saveToast = false), 2000);
  }

  function handleReset() {
    if (
      confirm("Are you sure you want to reset all AI configuration to default?")
    ) {
      resetAIConfig();
      provider = $aiConfigStore.provider;
      model = $aiConfigStore.model;
      apiKey = $aiConfigStore.apiKey;
      maxTokens = $aiConfigStore.maxTokens;
      customPrompt = $aiConfigStore.customPrompt;
      codeOnly = $aiConfigStore.codeOnly;

      resetToast = true;
      setTimeout(() => (resetToast = false), 2000);
    }
  }
</script>

<div class="max-w-[1400px] mx-auto p-4 sm:p-6 space-y-6">
  <div class="glass-panel p-6 sm:p-8 space-y-6">
    <!-- Header -->
    <div
      class="flex items-center justify-between flex-wrap gap-3 pb-4 border-b border-[var(--card-border)]"
    >
      <div
        class="flex items-center gap-2.5 text-lg font-extrabold text-[var(--accent-primary)]"
      >
        <i class="fa-solid fa-wand-magic-sparkles text-xl"></i>
        <span>AI Assistant Settings</span>
      </div>

      <div
        class="px-3.5 py-1.5 rounded-full text-xs font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 flex items-center gap-1.5"
      >
        <i class="fa-solid fa-robot"></i>
        <span
          >{apiKey && apiKey.length >= 5
            ? "Key Configured"
            : "Key Needed"}</span
        >
      </div>
    </div>

    <!-- Free Option Banner -->
    <div
      class="bg-indigo-500/10 border border-indigo-500/30 rounded-xl p-4 sm:p-5 flex items-start gap-3.5 text-xs sm:text-sm text-[var(--text-main)]"
    >
      <i
        class="fa-solid fa-circle-check text-indigo-400 text-lg shrink-0 mt-0.5"
      ></i>
      <div class="leading-relaxed">
        <strong class="font-extrabold text-[var(--text-main)]"
          >100% Free Option (No Credit Card Required):</strong
        >
        Select
        <strong class="text-indigo-400">OpenRouter (Free Tier)</strong> below!
        Get a free key instantly at
        <a
          href="https://openrouter.ai/keys"
          target="_blank"
          class="text-[var(--accent-cyan)] font-bold underline hover:text-cyan-300"
          >openrouter.ai/keys</a
        >
        with zero credit card setup.
      </div>
    </div>

    <!-- Form Inputs -->
    <div class="space-y-6">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- AI Provider -->
        <div class="space-y-2">
          <label
            for="providerSelect"
            class="text-xs sm:text-sm font-bold text-[var(--text-muted)] flex items-center gap-2"
          >
            <i class="fa-solid fa-server text-[var(--accent-primary)]"></i>
            <span>AI Provider:</span>
          </label>
          <select
            id="providerSelect"
            bind:value={provider}
            class="w-full px-4 py-3 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl text-xs sm:text-sm text-[var(--text-main)] outline-none focus:border-[var(--accent-primary)] cursor-pointer"
          >
            <option value="auto">Auto-Detect Provider (Recommended)</option>
            <option value="openrouter"
              >OpenRouter (Supports ALL Models: OpenAI, Claude, DeepSeek, Llama,
              Gemini)</option
            >
            <option value="openai">OpenAI (Direct: GPT-4o, GPT-4o-mini)</option>
            <option value="groq">Groq (Ultra-Fast Free Tier)</option>
            <option value="google">Google AI Studio (Gemini Direct Key)</option>
          </select>
        </div>

        <!-- Model Name -->
        <div class="space-y-2">
          <label
            for="modelInput"
            class="text-xs sm:text-sm font-bold text-[var(--text-muted)] flex items-center gap-2"
          >
            <i class="fa-solid fa-brain text-purple-400"></i>
            <span>Model Name:</span>
          </label>
          <input
            id="modelInput"
            type="text"
            list="modelSuggestions"
            bind:value={model}
            placeholder="e.g. openrouter/auto, openai/gpt-4o, claude-3.5-sonnet"
            class="w-full px-4 py-3 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl text-xs sm:text-sm text-[var(--text-main)] outline-none focus:border-[var(--accent-primary)]"
          />
          <datalist id="modelSuggestions">
            <option value="openrouter/auto"
              >OpenRouter Auto (Best Free Model)</option
            >
            <option value="openai/gpt-4o-mini">OpenAI GPT-4o Mini</option>
            <option value="openai/gpt-4o">OpenAI GPT-4o</option>
            <option value="anthropic/claude-3.5-sonnet"
              >Anthropic Claude 3.5 Sonnet</option
            >
            <option value="deepseek/deepseek-r1">DeepSeek R1</option>
            <option value="google/gemini-2.0-flash-exp:free"
              >Gemini 2.0 Flash (Free)</option
            >
            <option value="meta-llama/llama-3.2-11b-vision-instruct:free"
              >Llama 3.2 11B Vision (Free)</option
            >
            <option value="llama-3.2-11b-vision-preview"
              >Llama 3.2 11B (Groq)</option
            >
            <option value="gemini-2.0-flash"
              >Gemini 2.0 Flash (Google AI Studio)</option
            >
          </datalist>
        </div>
      </div>

      <!-- API Key with Eye Toggle -->
      <div class="space-y-2">
        <div class="flex justify-between items-center text-xs sm:text-sm">
          <label
            for="apiKeyInput"
            class="font-bold text-[var(--text-muted)] flex items-center gap-2"
          >
            <i class="fa-solid fa-key text-cyan-400"></i>
            <span>API Key:</span>
          </label>
          <a
            href="https://openrouter.ai/keys"
            target="_blank"
            class="text-[var(--accent-cyan)] hover:underline font-semibold text-xs"
          >
            Get key at openrouter.ai/keys
          </a>
        </div>

        <div class="relative">
          <input
            id="apiKeyInput"
            type={showApiKey ? "text" : "password"}
            bind:value={apiKey}
            placeholder="sk-or-v1-..."
            class="w-full px-4 py-3 pr-12 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl text-xs sm:text-sm font-mono text-[var(--text-main)] outline-none focus:border-[var(--accent-primary)]"
          />
          <button
            type="button"
            on:click={() => (showApiKey = !showApiKey)}
            class="absolute right-3.5 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--accent-primary)] transition-colors cursor-pointer"
            title={showApiKey ? "Hide API key" : "Show API key"}
          >
            <i
              class={showApiKey
                ? "fa-solid fa-eye-slash text-base"
                : "fa-solid fa-eye text-base"}
            ></i>
          </button>
        </div>
      </div>

      <!-- Max Output Tokens -->
      <div class="space-y-2">
        <label
          for="maxTokensInput"
          class="text-xs sm:text-sm font-bold text-[var(--text-muted)] flex items-center gap-2"
        >
          <i class="fa-solid fa-gauge-high text-amber-400"></i>
          <span>Max Output Tokens Limit:</span>
        </label>
        <input
          id="maxTokensInput"
          type="number"
          min="256"
          max="65536"
          step="256"
          bind:value={maxTokens}
          class="w-full px-4 py-3 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl text-xs sm:text-sm text-[var(--text-main)] outline-none focus:border-[var(--accent-primary)]"
        />
      </div>

      <!-- Code Only Output Checkbox -->
      <label
        for="codeOnlyCheck"
        class="flex items-center gap-3.5 p-4 bg-[var(--bg-tertiary)] border border-[var(--card-border)] rounded-xl cursor-pointer hover:bg-[var(--card-hover)] transition-colors"
      >
        <input
          id="codeOnlyCheck"
          type="checkbox"
          bind:checked={codeOnly}
          class="w-4 h-4 accent-[var(--accent-primary)] cursor-pointer shrink-0"
        />
        <span class="text-xs sm:text-sm font-semibold text-[var(--text-main)]">
          Clean Code Only Output (Automatically strips markdown text &amp;
          explanations)
        </span>
      </label>

      <!-- System Prompt Textarea -->
      <div class="space-y-2">
        <label
          for="sysPromptInput"
          class="text-xs sm:text-sm font-bold text-[var(--text-muted)] flex items-center gap-2"
        >
          <i class="fa-solid fa-terminal text-emerald-400"></i>
          <span>System Instruction Prompt:</span>
        </label>
        <textarea
          id="sysPromptInput"
          rows="6"
          bind:value={customPrompt}
          class="w-full p-4 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl text-xs sm:text-sm font-mono text-[var(--text-main)] outline-none focus:border-[var(--accent-primary)] resize-y custom-scrollbar min-h-[180px] leading-relaxed"
        ></textarea>
      </div>

      <!-- Advanced Relay Endpoint -->
      <div
        class="space-y-2 pt-3 border-t border-dashed border-[var(--card-border)]"
      >
        <label
          for="relayInput"
          class="text-xs sm:text-sm font-bold text-[var(--text-muted)] flex items-center gap-2"
        >
          <i class="fa-solid fa-network-wired text-orange-400"></i>
          <span>Relay Server Endpoint (Advanced / Self-Hosted):</span>
        </label>
        <input
          id="relayInput"
          type="text"
          bind:value={currentRelayUrl}
          placeholder="wss://ctrlv.onrender.com/ws"
          class="w-full px-4 py-3 bg-[var(--bg-input)] border border-[var(--card-border)] rounded-xl text-xs sm:text-sm font-mono text-[var(--text-main)] outline-none focus:border-[var(--accent-primary)]"
        />
        <div class="flex items-center justify-between flex-wrap gap-2 pt-1">
          <span class="text-xs text-[var(--text-muted)]">
            Default: <code class="text-cyan-400"
              >wss://ctrlv.onrender.com/ws</code
            > (Change only if self-hosting your own relay server)
          </span>
          <a
            href="https://github.com/Isu-Ismail/ctrlv/tree/main/relayserver"
            target="_blank"
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-[var(--bg-tertiary)] hover:bg-[var(--card-hover)] text-[var(--accent-primary)] hover:text-indigo-400 border border-[var(--card-border)] text-xs font-semibold transition-all text-decoration-none"
          >
            <i class="fa-brands fa-github"></i>
            <span>Self-Host Relay Server Source Code</span>
          </a>
        </div>
      </div>
    </div>

    <!-- Actions -->
    <div
      class="flex items-center justify-between flex-wrap gap-3 pt-4 border-t border-[var(--card-border)]"
    >
      <button
        type="button"
        on:click={handleReset}
        class="px-5 py-2.5 rounded-xl text-xs sm:text-sm font-bold text-rose-400 border border-rose-500/30 hover:bg-rose-500/10 transition-all cursor-pointer flex items-center gap-2"
      >
        <i class="fa-solid fa-rotate-left"></i>
        <span>Reset AI Settings</span>
      </button>

      <button
        type="button"
        on:click={handleSave}
        class="px-6 py-2.5 rounded-xl text-xs sm:text-sm font-bold bg-[var(--accent-primary)] hover:bg-[var(--accent-hover)] text-white shadow-lg shadow-indigo-500/25 transition-all cursor-pointer flex items-center gap-2"
      >
        {#if saveToast}
          <i class="fa-solid fa-check text-emerald-300"></i>
          <span>Saved Successfully!</span>
        {:else}
          <i class="fa-solid fa-floppy-disk"></i>
          <span>Save Settings</span>
        {/if}
      </button>
    </div>
  </div>
</div>
