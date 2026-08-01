import { getAIConfig, saveAIConfig, clearAIConfig } from "./config.js";

// UI Elements: Navigation & Views
const navBtnDashboard = document.getElementById("navBtnDashboard");
const navBtnDownload = document.getElementById("navBtnDownload");
const navBtnConfig = document.getElementById("navBtnConfig");
const navBtnHistory = document.getElementById("navBtnHistory");
const viewDashboard = document.getElementById("viewDashboard");
const viewDownload = document.getElementById("viewDownload");
const viewConfig = document.getElementById("viewConfig");
const viewHistory = document.getElementById("viewHistory");
const btnCardOpenHistory = document.getElementById("btnCardOpenHistory");
const btnClearAllHistory = document.getElementById("btnClearAllHistory");
const brandLogoBtn = document.getElementById("brandLogoBtn");

// UI Elements: Room & Header
const roomIdInput = document.getElementById("roomIdInput");
const btnConnect = document.getElementById("btnConnect");
const btnConnectText = document.getElementById("btnConnectText");
const connStatusBadge = document.getElementById("connStatusBadge");
const connStatusText = document.getElementById("connStatusText");
const clientCountsBadge = document.getElementById("clientCountsBadge");
const browserCount = document.getElementById("browserCount");
const pcCount = document.getElementById("pcCount");
const btnCopyRoom = document.getElementById("btnCopyRoom");
const btnCopyRoomId = document.getElementById("btnCopyRoomId");
const btnGenRandomRoom = document.getElementById("btnGenRandomRoom");

// UI Elements: Dashboard Sync
const screenshotImg = document.getElementById("screenshotImg");
const emptyState = document.getElementById("emptyState");
const textInput = document.getElementById("textInput");
const btnSendText = document.getElementById("btnSendText");
const imageModal = document.getElementById("imageModal");
const modalImg = document.getElementById("modalImg");
const btnCloseModal = document.getElementById("btnCloseModal");
const btnToggleAutoDownload = document.getElementById("btnToggleAutoDownload");
const autoDlLabel = document.getElementById("autoDlLabel");
const btnDownloadImg = document.getElementById("btnDownloadImg");

// UI Elements: AI Solver
const btnToggleAutoSolve = document.getElementById("btnToggleAutoSolve");
const autoSolveLabel = document.getElementById("autoSolveLabel");
const btnSolveGemini = document.getElementById("btnSolveGemini");
const aiInstructionInput = document.getElementById("aiInstructionInput");
const aiSolverStatusBadge = document.getElementById("aiSolverStatusBadge");
const aiSolverStatusText = document.getElementById("aiSolverStatusText");

// UI Elements: Tabs
const tabBtnScreenshot = document.getElementById("tabBtnScreenshot");
const tabBtnPCText = document.getElementById("tabBtnPCText");
const screenshotViewer = document.getElementById("screenshotViewer");
const pcSentTextViewer = document.getElementById("pcSentTextViewer");
const pcSentTextDisplay = document.getElementById("pcSentTextDisplay");
const screenshotActions = document.getElementById("screenshotActions");
const btnSolvePCSentText = document.getElementById("btnSolvePCSentText");

// UI Elements: Theme Switcher
const themeToggleBtn = document.getElementById("themeToggleBtn");
const themeToggleIcon = document.getElementById("themeToggleIcon");

// Initialize Theme from localStorage
const storedTheme = localStorage.getItem("ctrlv_theme") || "dark";
document.documentElement.setAttribute("data-theme", storedTheme);
if (themeToggleIcon) {
  themeToggleIcon.className = storedTheme === "dark" ? "fa-solid fa-moon" : "fa-solid fa-sun";
}

if (themeToggleBtn) {
  themeToggleBtn.addEventListener("click", () => {
    const currentTheme = document.documentElement.getAttribute("data-theme") || "dark";
    const newTheme = currentTheme === "light" ? "dark" : "light";
    document.documentElement.setAttribute("data-theme", newTheme);
    localStorage.setItem("ctrlv_theme", newTheme);
    if (themeToggleIcon) {
      themeToggleIcon.className = newTheme === "dark" ? "fa-solid fa-moon" : "fa-solid fa-sun";
    }
  });
}

// State Variables
let activeView = "dashboard";
let activeTab = "screenshot";
let ws = null;
let isConnected = false;
let autoDownload = localStorage.getItem("ctrlv_auto_download") === "true";
let autoSolveEnabled = localStorage.getItem("ctrlv_auto_solve") !== "false";
let currentRoomId = localStorage.getItem("ctrlv_room_id") || "ctrlv-a8f3b2";
let isSolvingAI = false;

// Cache variables to prevent laggy DOM re-renders
let cachedImagePath = null;
let cachedPCSentText = null;

function updateHistoryLinks() {
  const navHistory = document.getElementById("btnOpenHistoryNav");
  const cardHistory = document.getElementById("btnOpenHistoryCard");
  const url = `history.html?room=${encodeURIComponent(currentRoomId)}`;
  if (navHistory) navHistory.href = url;
  if (cardHistory) cardHistory.href = url;
}

if (roomIdInput) roomIdInput.value = currentRoomId;
updateHistoryLinks();

// Populate initial prompt from AI config
const savedAiConfig = getAIConfig();
if (aiInstructionInput && savedAiConfig.customPrompt) {
  aiInstructionInput.value = savedAiConfig.customPrompt;
}

// -----------------------------------------------------------------------------
// 1. SPA View Switching (Zero Page Reload = 100% Stable Connection Across Screens)
// -----------------------------------------------------------------------------
let selectedHistoryIdx = 0;

function escapeHtml(str) {
  if (!str) return "";
  return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#039;");
}

function renderHistoryView() {
  const sidebar = document.getElementById("historySidebar");
  const detail = document.getElementById("historyDetailPanel");
  if (!sidebar || !detail) return;

  const key = `ctrlv_history_${currentRoomId}`;
  let history = [];
  try {
    const stored = localStorage.getItem(key);
    history = stored ? JSON.parse(stored) : [];
  } catch (e) {
    history = [];
  }

  if (history.length === 0) {
    const cachedPC = getCachedPCText();
    const cachedWeb = getCachedWebText();
    const now = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    const today = new Date().toLocaleDateString();

    if (cachedPC && cachedPC.trim()) {
      history.push({ text: cachedPC.trim(), time: now, date: today });
    }
    if (cachedWeb && cachedWeb.trim() && cachedWeb.trim() !== cachedPC.trim()) {
      history.push({ text: cachedWeb.trim(), time: now, date: today });
    }

    if (history.length > 0) {
      localStorage.setItem(key, JSON.stringify(history));
    }
  }

  if (history.length === 0) {
    sidebar.innerHTML = `
      <div style="font-size:0.82rem; color:var(--text-muted); text-align:center; padding:1.5rem 0.5rem;">
        No history items
      </div>
    `;
    detail.innerHTML = `
      <div style="flex:1; display:flex; flex-direction:column; align-items:center; justify-content:center; padding:2rem; color:var(--text-muted); text-align:center;">
        <i class="fa-solid fa-folder-open" style="font-size:3rem; color:var(--text-muted); margin-bottom:1rem; opacity:0.5;"></i>
        <h3 style="font-weight:700; color:var(--text-main);">No History Available</h3>
        <p style="font-size:0.88rem; margin-top:0.5rem; max-width:320px;">Texts you send to your PC or questions received from your PC will be automatically saved here.</p>
      </div>
    `;
    return;
  }

  if (selectedHistoryIdx >= history.length) selectedHistoryIdx = 0;

  sidebar.innerHTML = history.map((item, idx) => {
    const snippet = (item.text || '').trim().split('\n')[0] || 'Empty Text';
    const isActive = idx === selectedHistoryIdx;
    return `
      <div class="history-item ${isActive ? 'active' : ''}" data-idx="${idx}" style="padding:0.65rem 0.75rem; border-radius:var(--radius-sm); cursor:pointer; background:${isActive ? 'var(--bg-secondary)' : 'transparent'}; border:1px solid ${isActive ? 'var(--accent-primary)' : 'transparent'}; transition:all 0.15s ease;">
        <div style="font-size:0.85rem; font-weight:600; color:var(--text-main); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; margin-bottom:0.2rem;">
          ${escapeHtml(snippet)}
        </div>
        <div style="font-size:0.75rem; color:var(--text-muted); display:flex; align-items:center; gap:0.35rem;">
          <i class="fa-regular fa-clock"></i> ${escapeHtml(item.time || 'Recent')} ${item.date ? ' &bull; ' + escapeHtml(item.date) : ''}
        </div>
      </div>
    `;
  }).join('');

  sidebar.querySelectorAll(".history-item").forEach(el => {
    el.addEventListener("click", () => {
      selectedHistoryIdx = parseInt(el.getAttribute("data-idx")) || 0;
      renderHistoryView();
    });
  });

  const selectedItem = history[selectedHistoryIdx];
  detail.innerHTML = `
    <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:0.5rem; padding-bottom:0.75rem; border-bottom:1px solid var(--card-border);">
      <div style="font-size:0.88rem; font-weight:700; color:var(--text-muted); display:flex; align-items:center; gap:0.5rem;">
        <i class="fa-solid fa-clock-rotate-left" style="color:var(--accent-primary);"></i>
        <span>Recorded: ${escapeHtml(selectedItem.time || 'Recent')} ${selectedItem.date ? ' (' + escapeHtml(selectedItem.date) + ')' : ''}</span>
      </div>

      <div style="display:flex; align-items:center; gap:0.5rem;">
        <button class="btn-send" id="btnCopySelectedHistory" style="padding:0.35rem 0.85rem; font-size:0.8rem;">
          <i class="fa-regular fa-copy"></i> <span id="copyHistoryLabel">Copy Text</span>
        </button>
        <button class="btn-tool" id="btnSendSelectedToPC" style="background:var(--accent-primary); color:white; border:none; padding:0.35rem 0.85rem; font-size:0.8rem; font-weight:700;">
          <i class="fa-solid fa-paper-plane"></i> Send to PC
        </button>
        <button class="btn-tool" id="btnDeleteSelectedHistory" style="color:var(--accent-red); padding:0.35rem 0.65rem;" title="Delete this entry">
          <i class="fa-solid fa-trash-can"></i>
        </button>
      </div>
    </div>
    <textarea class="code-textarea" id="historyDetailText" style="flex:1; width:100%; min-height:450px; margin-top:0.5rem;" readonly>${escapeHtml(selectedItem.text)}</textarea>
  `;

  const btnCopy = document.getElementById("btnCopySelectedHistory");
  if (btnCopy) {
    btnCopy.addEventListener("click", () => {
      navigator.clipboard.writeText(selectedItem.text).then(() => {
        const lbl = document.getElementById("copyHistoryLabel");
        if (lbl) lbl.textContent = "Copied!";
        setTimeout(() => { if (lbl) lbl.textContent = "Copy Text"; }, 1800);
      });
    });
  }

  const btnSendPC = document.getElementById("btnSendSelectedToPC");
  if (btnSendPC) {
    btnSendPC.addEventListener("click", () => {
      sendTextToRelay(selectedItem.text);
      btnSendPC.innerHTML = `<i class="fa-solid fa-check"></i> Sent!`;
      setTimeout(() => {
        btnSendPC.innerHTML = `<i class="fa-solid fa-paper-plane"></i> Send to PC`;
      }, 1800);
    });
  }

  const btnDelete = document.getElementById("btnDeleteSelectedHistory");
  if (btnDelete) {
    btnDelete.addEventListener("click", () => {
      history.splice(selectedHistoryIdx, 1);
      localStorage.setItem(key, JSON.stringify(history));
      renderHistoryView();
    });
  }
}

if (btnClearAllHistory) {
  btnClearAllHistory.addEventListener("click", () => {
    const key = `ctrlv_history_${currentRoomId}`;
    localStorage.removeItem(key);
    selectedHistoryIdx = 0;
    renderHistoryView();
  });
}

function switchView(targetView) {
  activeView = targetView;
  const views = {
    dashboard: viewDashboard,
    download: viewDownload,
    config: viewConfig,
    history: viewHistory
  };
  const navBtns = {
    dashboard: navBtnDashboard,
    download: navBtnDownload,
    config: navBtnConfig,
    history: navBtnHistory
  };

  Object.keys(views).forEach(v => {
    if (views[v]) views[v].style.display = (v === targetView) ? "block" : "none";
    if (navBtns[v]) navBtns[v].className = (v === targetView) ? "nav-tab-btn active" : "nav-tab-btn";
  });

  if (targetView === "config") {
    populateConfigUI();
  } else if (targetView === "history") {
    renderHistoryView();
  }
}

if (navBtnDashboard) navBtnDashboard.addEventListener("click", () => switchView("dashboard"));
if (navBtnDownload) navBtnDownload.addEventListener("click", () => switchView("download"));
if (navBtnConfig) navBtnConfig.addEventListener("click", () => switchView("config"));
if (navBtnHistory) navBtnHistory.addEventListener("click", () => switchView("history"));
if (btnCardOpenHistory) btnCardOpenHistory.addEventListener("click", () => switchView("history"));
if (brandLogoBtn) brandLogoBtn.addEventListener("click", (e) => { e.preventDefault(); switchView("dashboard"); });

// Tab Switching inside Dashboard
function switchTab(tab) {
  activeTab = tab;
  const btnDownloadImg = document.getElementById("btnDownloadImg");
  const btnCopyPCSentText = document.getElementById("btnCopyPCSentText");

  if (tab === "screenshot") {
    if (tabBtnScreenshot) tabBtnScreenshot.className = "tab-btn active";
    if (tabBtnPCText) tabBtnPCText.className = "tab-btn";
    if (screenshotViewer) screenshotViewer.style.display = "flex";
    if (pcSentTextViewer) pcSentTextViewer.style.display = "none";
    if (btnDownloadImg) btnDownloadImg.style.display = "inline-flex";
    if (btnCopyPCSentText) btnCopyPCSentText.style.display = "none";
  } else {
    if (tabBtnScreenshot) tabBtnScreenshot.className = "tab-btn";
    if (tabBtnPCText) tabBtnPCText.className = "tab-btn active";
    if (screenshotViewer) screenshotViewer.style.display = "none";
    if (pcSentTextViewer) pcSentTextViewer.style.display = "flex";
    if (btnDownloadImg) btnDownloadImg.style.display = "none";
    if (btnCopyPCSentText) btnCopyPCSentText.style.display = "inline-flex";
  }
}

if (tabBtnScreenshot) tabBtnScreenshot.addEventListener("click", () => switchTab("screenshot"));
if (tabBtnPCText) tabBtnPCText.addEventListener("click", () => switchTab("pctext"));

// -----------------------------------------------------------------------------
// 2. Persistent Local Storage Cache & Image Viewing
// -----------------------------------------------------------------------------
function getCachedScreenshot() {
  try {
    return localStorage.getItem(`ctrlv_last_screenshot_${currentRoomId}`) || null;
  } catch (e) {
    return null;
  }
}

function saveCachedScreenshot(b64Data) {
  if (!b64Data || b64Data.trim() === "") return;
  try {
    localStorage.setItem(`ctrlv_last_screenshot_${currentRoomId}`, b64Data);
  } catch (e) {
    console.warn("Storage full when caching screenshot:", e);
  }
}

function getCachedWebText() {
  try {
    return localStorage.getItem(`ctrlv_last_web_text_${currentRoomId}`) || "";
  } catch (e) {
    return "";
  }
}

function saveCachedWebText(text) {
  if (text === undefined || text === null) return;
  try {
    localStorage.setItem(`ctrlv_last_web_text_${currentRoomId}`, text);
  } catch (e) {}
}

function getCachedPCText() {
  try {
    return localStorage.getItem(`ctrlv_last_pc_text_${currentRoomId}`) || "";
  } catch (e) {
    return "";
  }
}

function saveCachedPCText(text) {
  if (text === undefined || text === null) return;
  try {
    localStorage.setItem(`ctrlv_last_pc_text_${currentRoomId}`, text);
  } catch (e) {}
}

function loadAndDisplayCachedData() {
  const cached = getCachedScreenshot();
  if (cached && cached.trim() !== "") {
    cachedImagePath = cached;
    if (screenshotImg) {
      screenshotImg.src = cached;
      screenshotImg.style.display = "block";
    }
    if (emptyState) emptyState.style.display = "none";
  }

  const cachedWeb = getCachedWebText();
  if (textInput && cachedWeb) {
    textInput.value = cachedWeb;
  }

  const cachedPC = getCachedPCText();
  if (cachedPC && cachedPC.trim() !== "") {
    cachedPCSentText = cachedPC;
    if (pcSentTextDisplay) {
      pcSentTextDisplay.value = cachedPC;
    }
  }
}

// Initial UI Toggle Displays
updateAutoDownloadUI();
updateAutoSolveUI();

function updateAutoDownloadUI() {
  if (!btnToggleAutoDownload) return;
  btnToggleAutoDownload.className = autoDownload ? "btn-tool active" : "btn-tool";
  autoDlLabel.textContent = autoDownload ? "Auto-Save: ON" : "Auto-Save: OFF";
}

if (btnToggleAutoDownload) {
  btnToggleAutoDownload.addEventListener("click", () => {
    autoDownload = !autoDownload;
    localStorage.setItem("ctrlv_auto_download", autoDownload ? "true" : "false");
    updateAutoDownloadUI();
  });
}

function updateAutoSolveUI() {
  if (!btnToggleAutoSolve) return;
  btnToggleAutoSolve.className = autoSolveEnabled ? "btn-tool active" : "btn-tool";
  if (autoSolveLabel) autoSolveLabel.textContent = autoSolveEnabled ? "Auto-Solve: ON" : "Auto-Solve: OFF";
}

if (btnToggleAutoSolve) {
  btnToggleAutoSolve.addEventListener("click", () => {
    autoSolveEnabled = !autoSolveEnabled;
    localStorage.setItem("ctrlv_auto_solve", autoSolveEnabled ? "true" : "false");
    updateAutoSolveUI();
  });
}

function updateAISolverStatus(state, msg) {
  const badges = [
    { badge: document.getElementById("aiSolverStatusBadge"), text: document.getElementById("aiSolverStatusText") },
    { badge: document.getElementById("aiStatusBadge"), text: document.getElementById("aiStatusText") }
  ];

  badges.forEach(({ badge, text }) => {
    if (!badge || !text) return;
    if (state === "solving") {
      badge.className = "status-pill pending";
      badge.style.background = "#eff6ff";
      badge.style.color = "#2563eb";
      badge.style.borderColor = "#93c5fd";
      text.innerHTML = `<i class="fa-solid fa-spinner fa-spin"></i> ${msg || "AI Solving..."}`;
    } else if (state === "success") {
      badge.className = "status-pill seen";
      badge.style.background = "";
      badge.style.color = "";
      badge.style.borderColor = "";
      text.innerHTML = `<i class="fa-solid fa-circle-check"></i> ${msg || "Text Received & Pushed to PC!"}`;
    } else if (state === "ready") {
      badge.className = "status-pill seen";
      badge.style.background = "#ecfdf5";
      badge.style.color = "#059669";
      badge.style.borderColor = "#a7f3d0";
      text.innerHTML = `<i class="fa-solid fa-robot"></i> ${msg || "AI Ready"}`;
    } else if (state === "error") {
      badge.className = "status-pill offline";
      badge.style.background = "#fef2f2";
      badge.style.color = "#dc2626";
      badge.style.borderColor = "#fca5a5";
      text.innerHTML = `<i class="fa-solid fa-triangle-exclamation"></i> ${msg || "AI Key Missing"}`;
    } else {
      badge.className = "status-pill pending";
      badge.style.background = "";
      badge.style.color = "";
      badge.style.borderColor = "";
      text.innerHTML = `<i class="fa-solid fa-robot"></i> ${msg || "AI Ready"}`;
    }
  });
}

function checkAIKeyStatus() {
  const cfg = getAIConfig();
  if (cfg && cfg.apiKey && cfg.apiKey.trim().length >= 5) {
    updateAISolverStatus("ready", "AI Ready");
  } else {
    updateAISolverStatus("error", "AI Key Missing");
  }
}

// Initial AI Key Status Check
checkAIKeyStatus();

// -----------------------------------------------------------------------------
// Error Alert Modal Popup (Glassmorphic Window with OK / Cancel Controls)
// -----------------------------------------------------------------------------
const errorModal = document.getElementById("errorModal");
const errorModalTitle = document.getElementById("errorModalTitle");
const errorModalBody = document.getElementById("errorModalBody");
const btnErrorOk = document.getElementById("btnErrorOk");
const btnErrorCancel = document.getElementById("btnErrorCancel");

function showErrorModal(title, msg) {
  if (errorModalTitle) errorModalTitle.textContent = title || "API Provider Error";
  if (errorModalBody) errorModalBody.textContent = msg || "An unknown error occurred.";
  if (errorModal) errorModal.style.display = "flex";
}

function hideErrorModal() {
  if (errorModal) errorModal.style.display = "none";
}

if (btnErrorOk) btnErrorOk.addEventListener("click", hideErrorModal);
if (btnErrorCancel) btnErrorCancel.addEventListener("click", hideErrorModal);
if (errorModal) {
  errorModal.addEventListener("click", (e) => {
    if (e.target === errorModal) hideErrorModal();
  });
}
window.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && errorModal && errorModal.style.display === "flex") {
    hideErrorModal();
  }
});

function resolveAIProvider(provider, apiKey) {
  if (provider && provider !== "auto") return provider;
  if (apiKey.startsWith("gsk_")) return "groq";
  if (apiKey.startsWith("AIza")) return "google";
  return "openrouter"; // Default to OpenRouter for OpenAI, Claude, DeepSeek, Llama, Qwen, Gemini, etc.
}

// -----------------------------------------------------------------------------
// 3. Multi-Provider Vision AI Solver (OpenRouter / Groq / Google AI Studio)
// -----------------------------------------------------------------------------
async function solveImageWithGemini(b64ImageData, promptText) {
  if (isSolvingAI) return;

  const aiConfig = getAIConfig();
  if (!aiConfig || !aiConfig.apiKey) {
    updateAISolverStatus("error", "Configure AI Key in Config tab first!");
    return;
  }

  const cleanApiKey = aiConfig.apiKey.replace(/["'\s]/g, "");
  if (!cleanApiKey || cleanApiKey.length < 5) {
    updateAISolverStatus("error", "AI API Key is invalid or empty!");
    return;
  }

  if (!b64ImageData || b64ImageData.trim() === "") {
    updateAISolverStatus("error", "No screenshot available to solve");
    return;
  }

  isSolvingAI = true;
  const provider = resolveAIProvider(aiConfig.provider, cleanApiKey);
  updateAISolverStatus("solving", `${provider.toUpperCase()} Analyzing Screenshot...`);

  let fullB64Url = b64ImageData;
  if (!fullB64Url.startsWith("data:")) {
    fullB64Url = "data:image/jpeg;base64," + fullB64Url;
  }

  const prompt = promptText || aiConfig.customPrompt || "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting.";

  try {
    let generatedText = "";

    if (provider === "openrouter") {
      const model = aiConfig.model || "openrouter/auto";
      const response = await fetch("https://openrouter.ai/api/v1/chat/completions", {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${cleanApiKey}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          model: model,
          max_tokens: parseInt(aiConfig.maxTokens) || 2048,
          messages: [
            {
              role: "user",
              content: [
                { type: "text", text: prompt },
                { type: "image_url", image_url: { url: fullB64Url } }
              ]
            }
          ]
        })
      });

      if (!response.ok) {
        const errJson = await response.json().catch(() => ({}));
        throw new Error(errJson?.error?.message || `OpenRouter Error ${response.status}`);
      }

      const resData = await response.json();
      generatedText = resData?.choices?.[0]?.message?.content;

    } else if (provider === "groq") {
      const model = aiConfig.model || "llama-3.2-11b-vision-preview";
      const response = await fetch("https://api.groq.com/openai/v1/chat/completions", {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${cleanApiKey}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          model: model,
          messages: [
            {
              role: "user",
              content: [
                { type: "text", text: prompt },
                { type: "image_url", image_url: { url: fullB64Url } }
              ]
            }
          ]
        })
      });

      if (!response.ok) {
        const errJson = await response.json().catch(() => ({}));
        throw new Error(errJson?.error?.message || `Groq Error ${response.status}`);
      }

      const resData = await response.json();
      generatedText = resData?.choices?.[0]?.message?.content;

    } else {
      const model = aiConfig.model || "gemini-2.0-flash";
      const endpoint = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent?key=${cleanApiKey}`;

      let cleanB64 = b64ImageData;
      let mimeType = "image/jpeg";
      if (cleanB64.startsWith("data:")) {
        const parts = cleanB64.split(";base64,");
        if (parts.length === 2) {
          mimeType = parts[0].replace("data:", "");
          cleanB64 = parts[1];
        }
      }

      const response = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          contents: [{
            parts: [
              { text: prompt },
              { inline_data: { mime_type: mimeType, data: cleanB64 } }
            ]
          }]
        })
      });

      if (!response.ok) {
        const errJson = await response.json().catch(() => ({}));
        throw new Error(errJson?.error?.message || `Google Error ${response.status}`);
      }

      const resData = await response.json();
      generatedText = resData?.candidates?.[0]?.content?.parts?.[0]?.text;
    }

    if (!generatedText || generatedText.trim() === "") {
      throw new Error("AI returned empty response");
    }

    if (aiConfig.codeOnly !== false) {
      generatedText = parseCleanCodeOnly(generatedText);
    }

    textInput.value = generatedText;
    sendTextToRelay(generatedText);
    updateAISolverStatus("success", "Text Received & Pushed to PC!");
  } catch (err) {
    console.error("AI Solver error:", err);
    updateAISolverStatus("error", "AI Error");
    showErrorModal("API Provider Error", err.message || "Failed to solve with AI");
  } finally {
    isSolvingAI = false;
  }
}

async function solveTextWithGemini(questionText, promptText) {
  if (isSolvingAI) return;

  const aiConfig = getAIConfig();
  if (!aiConfig || !aiConfig.apiKey) {
    updateAISolverStatus("error", "Configure AI Key in Config tab first!");
    return;
  }

  const cleanApiKey = aiConfig.apiKey.replace(/["'\s]/g, "");
  if (!cleanApiKey || cleanApiKey.length < 5) {
    updateAISolverStatus("error", "AI API Key is invalid or empty!");
    return;
  }

  if (!questionText || questionText.trim() === "") {
    updateAISolverStatus("error", "No question text available to solve");
    return;
  }

  isSolvingAI = true;
  const provider = resolveAIProvider(aiConfig.provider, cleanApiKey);
  updateAISolverStatus("solving", `${provider.toUpperCase()} Solving Text Question...`);

  const userPrompt = promptText || aiConfig.customPrompt || "Solve the problem shown in this text. Output ONLY clean, working code without explanations or markdown formatting.";
  const fullTextPrompt = `${userPrompt}\n\nQuestion Context / Problem Statement:\n${questionText}`;

  try {
    let generatedText = "";
    if (provider === "openrouter") {
      const model = aiConfig.model || "openrouter/auto";
      const resp = await fetch("https://openrouter.ai/api/v1/chat/completions", {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${cleanApiKey}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          model: model,
          max_tokens: parseInt(aiConfig.maxTokens) || 2048,
          messages: [{ role: "user", content: fullTextPrompt }]
        })
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`OpenRouter HTTP ${resp.status}: ${errText}`);
      }
      const data = await resp.json();
      generatedText = data.choices?.[0]?.message?.content || "";
    } else if (provider === "groq") {
      const model = aiConfig.model || "llama-3.2-11b-vision-preview";
      const resp = await fetch("https://api.groq.com/openai/v1/chat/completions", {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${cleanApiKey}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          model: model,
          messages: [{ role: "user", content: fullTextPrompt }]
        })
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`Groq HTTP ${resp.status}: ${errText}`);
      }
      const data = await resp.json();
      generatedText = data.choices?.[0]?.message?.content || "";
    } else {
      const model = aiConfig.model || "gemini-2.0-flash";
      const endpoint = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent?key=${cleanApiKey}`;
      const resp = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          contents: [{ parts: [{ text: fullTextPrompt }] }]
        })
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`Gemini HTTP ${resp.status}: ${errText}`);
      }
      const data = await resp.json();
      generatedText = data.candidates?.[0]?.content?.parts?.[0]?.text || "";
    }

    if (!generatedText || generatedText.trim() === "") {
      throw new Error("AI returned empty response");
    }

    if (aiConfig.codeOnly !== false) {
      generatedText = parseCleanCodeOnly(generatedText);
    }

    textInput.value = generatedText;
    sendTextToRelay(generatedText);
    updateAISolverStatus("success", "Text Received & Pushed to PC!");
  } catch (err) {
    console.error("AI Text Solver error:", err);
    updateAISolverStatus("error", "AI Error");
    showErrorModal("API Provider Error", err.message || "AI Solving failed");
  } finally {
    isSolvingAI = false;
  }
}

function parseCleanCodeOnly(rawText) {
  if (!rawText) return "";
  const codeBlockRegex = /```(?:[a-zA-Z0-9_+-]+)?\n([\s\S]*?)```/g;
  const matches = [...rawText.matchAll(codeBlockRegex)];
  if (matches.length > 0) {
    return matches.map(m => m[1].trim()).join("\n\n");
  }
  return rawText.trim();
}

function saveToRoomHistory(text) {
  if (!text || !text.trim()) return;
  try {
    const key = `ctrlv_history_${currentRoomId}`;
    const existing = localStorage.getItem(key);
    let history = existing ? JSON.parse(existing) : [];

    const clean = text.trim();
    if (history.length > 0 && history[0].text === clean) {
      return;
    }

    const newItem = {
      text: clean,
      time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      date: new Date().toLocaleDateString()
    };

    history.unshift(newItem);

    if (history.length > 50) {
      history = history.slice(0, 50);
    }

    localStorage.setItem(key, JSON.stringify(history));
  } catch (e) {
    console.warn("Failed to update room history:", e);
  }
}

function sendTextToRelay(cleanText) {
  if (!cleanText) return;
  saveCachedWebText(cleanText);
  saveToRoomHistory(cleanText);
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: "web_exe", room_id: currentRoomId, content: cleanText, sender_id: "browser" }));
  }
}

if (btnSolveGemini) {
  btnSolveGemini.addEventListener("click", () => {
    const prompt = aiInstructionInput ? aiInstructionInput.value.trim() : "";
    if (activeTab === "pctext") {
      const question = pcSentTextDisplay ? pcSentTextDisplay.value.trim() : "";
      if (!question || question.startsWith("No question text received")) {
        updateAISolverStatus("error", "No question text available to solve!");
        return;
      }
      solveTextWithGemini(question, prompt);
    } else {
      const b64Data = screenshotImg.src;
      if (!b64Data || b64Data === "" || screenshotImg.style.display === "none") {
        updateAISolverStatus("error", "No screenshot image to solve!");
        return;
      }
      solveImageWithGemini(b64Data, prompt);
    }
  });
}

if (btnSolvePCSentText) {
  btnSolvePCSentText.addEventListener("click", () => {
    const question = pcSentTextDisplay ? pcSentTextDisplay.value.trim() : "";
    if (!question || question.startsWith("No question text received")) {
      updateAISolverStatus("error", "No question text available to solve!");
      return;
    }
    const prompt = aiInstructionInput ? aiInstructionInput.value.trim() : "";
    solveTextWithGemini(question, prompt);
  });
}

const btnCopyPCSentText = document.getElementById("btnCopyPCSentText");
if (btnCopyPCSentText) {
  btnCopyPCSentText.addEventListener("click", () => {
    const text = pcSentTextDisplay ? pcSentTextDisplay.value : "";
    if (text) {
      navigator.clipboard.writeText(text).then(() => {
        const orig = btnCopyPCSentText.innerHTML;
        btnCopyPCSentText.innerHTML = `<i class="fa-solid fa-check"></i> Copied!`;
        setTimeout(() => { btnCopyPCSentText.innerHTML = orig; }, 2000);
      });
    }
  });
}

// -----------------------------------------------------------------------------
// 4. WebSocket Connection Engine (Manual Connect Button Control & NO Auto-Connect on Load)
// -----------------------------------------------------------------------------
function updateConnectionStatus(online) {
  isConnected = online;
  if (!connStatusBadge) return;
  if (online) {
    connStatusBadge.className = "conn-status-badge online";
    connStatusText.textContent = "Connected";
    if (btnConnect) {
      btnConnect.disabled = false;
      btnConnect.className = "btn-create-room connected";
    }
    if (btnConnectText) btnConnectText.innerHTML = "Disconnect";
    if (clientCountsBadge) clientCountsBadge.style.display = "inline-flex";
    if (browserCount && (parseInt(browserCount.textContent || "0") < 1)) {
      browserCount.textContent = "1";
    }
  } else {
    connStatusBadge.className = "conn-status-badge offline";
    connStatusText.textContent = "Disconnected";
    if (btnConnect) {
      btnConnect.disabled = false;
      btnConnect.className = "btn-create-room";
    }
    if (btnConnectText) btnConnectText.innerHTML = "Connect";
    if (clientCountsBadge) clientCountsBadge.style.display = "none";
  }
}

function connectToRoom(roomId) {
  if (ws) {
    try { ws.close(); } catch (e) {}
  }

  currentRoomId = roomId;
  localStorage.setItem("ctrlv_room_id", roomId);
  updateHistoryLinks();

  cachedImagePath = null;
  cachedPCSentText = null;

  loadAndDisplayCachedData();

  if (btnConnect) {
    btnConnect.disabled = true;
  }
  if (btnConnectText) {
    btnConnectText.innerHTML = `<i class="fa-solid fa-spinner fa-spin"></i> Connecting...`;
  }

  const savedRelayUrl = (localStorage.getItem("ctrlv_relay_url") || "wss://ctrlv.onrender.com/ws").trim();
  const fullWsUrl = `${savedRelayUrl}?room=${encodeURIComponent(roomId)}&client=browser`;

  try {
    ws = new WebSocket(fullWsUrl);

    ws.onopen = () => {
      updateConnectionStatus(true);
      console.log(`Connected to WebSocket Relay Room: ${roomId}`);
    };

    ws.onclose = () => {
      updateConnectionStatus(false);
    };

    ws.onerror = (err) => {
      console.error("WebSocket Error:", err);
      updateConnectionStatus(false);
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === "room_stats") {
          if (browserCount) browserCount.textContent = Math.max(1, msg.browsers || 0);
          if (pcCount) pcCount.textContent = msg.clis || 0;
        } else if (msg.type === "image" && msg.content) {
          const newImg = msg.content;
          const isFreshImage = (newImg !== cachedImagePath);
          
          cachedImagePath = newImg;
          screenshotImg.src = newImg;
          screenshotImg.style.display = "block";
          if (emptyState) emptyState.style.display = "none";
          saveCachedScreenshot(newImg);

          if (isFreshImage) {
            if (autoDownload) {
              downloadImage(newImg, `ctrlv-${currentRoomId}-${Date.now()}.png`);
            }
          }

          if (autoSolveEnabled && isFreshImage) {
            switchTab("screenshot");
            const prompt = aiInstructionInput ? aiInstructionInput.value.trim() : "";
            solveImageWithGemini(newImg, prompt);
          }
        } else if ((msg.type === "exe_web" || msg.type === "text" || msg.type === "web_exe") && msg.content) {
          const newText = msg.content;
          if (msg.type === "web_exe") {
            if (textInput && textInput.value !== newText) {
              textInput.value = newText;
            }
            saveCachedWebText(newText);
          } else if (msg.type === "exe_web" || msg.type === "text") {
            const isFreshText = (newText !== cachedPCSentText);
            cachedPCSentText = newText;

            if (pcSentTextDisplay) {
              pcSentTextDisplay.value = newText;
            }
            saveCachedPCText(newText);
            saveToRoomHistory(newText);

            if (autoSolveEnabled && isFreshText) {
              switchTab("pctext");
              const prompt = aiInstructionInput ? aiInstructionInput.value.trim() : "";
              solveTextWithGemini(newText, prompt);
            }
          }
        }
      } catch (e) {
        console.error("Failed to parse WebSocket message:", e);
      }
    };
  } catch (err) {
    console.error("WebSocket Connection Error:", err);
    updateConnectionStatus(false);
  }
}

// Generate Random Room ID helper
function generateRandomRoomId() {
  const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
  let randomStr = "";
  for (let i = 0; i < 6; i++) {
    randomStr += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return `ctrlv-${randomStr}`;
}

if (btnGenRandomRoom) {
  btnGenRandomRoom.addEventListener("click", () => {
    const newRoom = generateRandomRoomId();
    if (roomIdInput) roomIdInput.value = newRoom;
    if (isConnected) {
      connectToRoom(newRoom);
    }
    // Auto-copy generated room ID to clipboard
    navigator.clipboard.writeText(newRoom).then(() => {
      const originalText = btnGenRandomRoom.innerHTML;
      btnGenRandomRoom.innerHTML = `<i class="fa-solid fa-check"></i>`;
      setTimeout(() => {
        btnGenRandomRoom.innerHTML = originalText;
      }, 1800);
    }).catch(e => console.warn("Clipboard access error:", e));
  });
}

if (btnCopyRoomId) {
  btnCopyRoomId.addEventListener("click", () => {
    const room = roomIdInput ? roomIdInput.value.trim() : "ctrlv-a8f3b2";
    if (!room) return;
    navigator.clipboard.writeText(room).then(() => {
      const originalText = btnCopyRoomId.innerHTML;
      btnCopyRoomId.innerHTML = `<i class="fa-solid fa-check"></i>`;
      setTimeout(() => {
        btnCopyRoomId.innerHTML = originalText;
      }, 1800);
    });
  });
}

// Connect / Disconnect Toggle Button
btnConnect.addEventListener("click", () => {
  if (isConnected) {
    if (ws) {
      try { ws.close(); } catch (e) {}
    }
    updateConnectionStatus(false);
  } else {
    const room = roomIdInput ? roomIdInput.value.trim() : "";
    if (room) {
      connectToRoom(room);
    }
  }
});

btnSendText.addEventListener("click", () => {
  const content = textInput ? textInput.value.trim() : "";
  if (!content) return;
  sendTextToRelay(content);

  const originalHTML = btnSendText.innerHTML;
  btnSendText.innerHTML = `<i class="fa-solid fa-check"></i> Sent!`;
  btnSendText.style.background = "var(--accent-green)";
  setTimeout(() => {
    btnSendText.innerHTML = originalHTML;
    btnSendText.style.background = "";
  }, 1800);
});

btnCopyRoom.addEventListener("click", () => {
  const room = roomIdInput ? roomIdInput.value.trim() : "ctrlv-a8f3b2";
  const cmd = `ctrlv -r ${room} -s`;
  navigator.clipboard.writeText(cmd).then(() => {
    const originalText = btnCopyRoom.innerHTML;
    btnCopyRoom.innerHTML = `<i class="fa-solid fa-check"></i> Copied!`;
    setTimeout(() => {
      btnCopyRoom.innerHTML = originalText;
    }, 2000);
  });
});

function downloadImage(dataUrl, filename) {
  const a = document.createElement("a");
  a.href = dataUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}

if (btnDownloadImg) {
  btnDownloadImg.addEventListener("click", () => {
    if (screenshotImg.src && screenshotImg.style.display !== "none") {
      downloadImage(screenshotImg.src, `ctrlv-${currentRoomId}-${Date.now()}.png`);
    }
  });
}

if (screenshotImg) {
  screenshotImg.addEventListener("click", () => {
    modalImg.src = screenshotImg.src;
    imageModal.style.display = "flex";
  });
}

if (btnCloseModal) {
  btnCloseModal.addEventListener("click", () => {
    imageModal.style.display = "none";
  });
}

// -----------------------------------------------------------------------------
// Config Form Manager (Load, Save, Reset, Key Visibility Toggle)
// -----------------------------------------------------------------------------
const aiProviderSelect = document.getElementById("aiProviderSelect");
const aiModelInput = document.getElementById("aiModelInput");
const aiApiKeyInput = document.getElementById("aiApiKeyInput");
const aiMaxTokensInput = document.getElementById("aiMaxTokensInput");
const aiCodeOnlyCheck = document.getElementById("aiCodeOnlyCheck");
const aiPromptInput = document.getElementById("aiPromptInput");
const relayUrlInput = document.getElementById("relayUrlInput");
const btnSaveAI = document.getElementById("btnSaveAI");
const btnResetAI = document.getElementById("btnResetAI");
const btnToggleKeyVis = document.getElementById("btnToggleKeyVis");
const eyeIcon = document.getElementById("eyeIcon");

function populateConfigUI() {
  const cfg = getAIConfig();
  if (aiProviderSelect) aiProviderSelect.value = cfg.provider || "auto";
  if (aiModelInput) aiModelInput.value = cfg.model || "openrouter/auto";
  if (aiApiKeyInput) aiApiKeyInput.value = cfg.apiKey || "";
  if (aiMaxTokensInput) aiMaxTokensInput.value = cfg.maxTokens || 2048;
  if (aiCodeOnlyCheck) aiCodeOnlyCheck.checked = cfg.codeOnly !== false;
  if (aiPromptInput) aiPromptInput.value = cfg.customPrompt || "";

  const savedRelay = localStorage.getItem("ctrlv_relay_url") || "wss://ctrlv.onrender.com/ws";
  if (relayUrlInput) relayUrlInput.value = savedRelay;
}

// Populate fields on startup
populateConfigUI();

if (btnToggleKeyVis) {
  btnToggleKeyVis.addEventListener("click", () => {
    if (!aiApiKeyInput) return;
    const isPass = aiApiKeyInput.type === "password";
    aiApiKeyInput.type = isPass ? "text" : "password";
    if (eyeIcon) {
      eyeIcon.className = isPass ? "fa-solid fa-eye-slash" : "fa-solid fa-eye";
    }
  });
}

if (btnSaveAI) {
  btnSaveAI.addEventListener("click", () => {
    const provider = aiProviderSelect ? aiProviderSelect.value : "auto";
    const model = aiModelInput ? aiModelInput.value.trim() : "openrouter/auto";
    const apiKey = aiApiKeyInput ? aiApiKeyInput.value.trim() : "";
    const maxTokens = aiMaxTokensInput ? parseInt(aiMaxTokensInput.value) || 2048 : 2048;
    const codeOnly = aiCodeOnlyCheck ? aiCodeOnlyCheck.checked : true;
    const customPrompt = aiPromptInput ? aiPromptInput.value.trim() : "";
    const relayUrl = relayUrlInput ? relayUrlInput.value.trim() : "";

    saveAIConfig(provider, apiKey, model, codeOnly, customPrompt, maxTokens);

    if (relayUrl) {
      localStorage.setItem("ctrlv_relay_url", relayUrl);
    }

    if (aiInstructionInput && customPrompt) {
      aiInstructionInput.value = customPrompt;
    }

    checkAIKeyStatus();

    const origHTML = btnSaveAI.innerHTML;
    btnSaveAI.innerHTML = `<i class="fa-solid fa-check"></i> Settings Saved!`;
    btnSaveAI.style.background = "var(--accent-green)";
    setTimeout(() => {
      btnSaveAI.innerHTML = origHTML;
      btnSaveAI.style.background = "";
    }, 2000);
  });
}

if (btnResetAI) {
  btnResetAI.addEventListener("click", () => {
    clearAIConfig();
    populateConfigUI();
    checkAIKeyStatus();
    const origHTML = btnResetAI.innerHTML;
    btnResetAI.innerHTML = `<i class="fa-solid fa-check"></i> Reset!`;
    setTimeout(() => {
      btnResetAI.innerHTML = origHTML;
    }, 2000);
  });
}

// -----------------------------------------------------------------------------
// 5. CLI Downloads Panel Script logic (Windows/Linux/macOS + Arch Pill Handlers)
// -----------------------------------------------------------------------------
const osBtns = document.querySelectorAll(".dl-os-btn");
const archBtns = document.querySelectorAll(".dl-arch-pill");
const panels = {
  windows: document.getElementById("panelWindows"),
  linux: document.getElementById("panelLinux"),
  mac: document.getElementById("panelMac")
};

let currentOs = "windows";
let currentArch = "amd64";

function updateDlContent() {
  if (!panels.windows) return;
  Object.keys(panels).forEach(os => {
    if (panels[os]) panels[os].style.display = (os === currentOs) ? "block" : "none";
  });

  osBtns.forEach(btn => {
    btn.classList.toggle("active", btn.dataset.os === currentOs);
  });

  archBtns.forEach(btn => {
    btn.classList.toggle("active", btn.dataset.arch === currentArch);
  });

  const archLabel = (currentArch === "amd64") ? "x64" : "ARM64";

  const winZipUrl = `https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_windows_${currentArch}.zip`;
  const elPs = document.getElementById("winPsCmd");
  if (elPs) elPs.textContent = `Invoke-WebRequest -Uri ${winZipUrl} -OutFile ctrlv.zip; Expand-Archive ctrlv.zip -DestinationPath C:\\ctrlv`;
  const elZipLink = document.getElementById("winZipLink");
  if (elZipLink) elZipLink.href = winZipUrl;
  const elZipText = document.getElementById("winZipBtnText");
  if (elZipText) elZipText.textContent = `Download Portable ZIP (${archLabel})`;

  const linuxDebUrl = `https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_linux_${currentArch}.deb`;
  const linuxRpmUrl = `https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_linux_${currentArch}.rpm`;
  const elApt = document.getElementById("linuxAptCmd");
  if (elApt) elApt.textContent = `wget ${linuxDebUrl} && sudo apt install ./ctrlv-cli_linux_${currentArch}.deb`;
  const elDebLink = document.getElementById("linuxDebLink");
  if (elDebLink) elDebLink.href = linuxDebUrl;
  const elDebText = document.getElementById("linuxDebBtnText");
  if (elDebText) elDebText.textContent = `Download .deb (${archLabel})`;
  const elRpmLink = document.getElementById("linuxRpmLink");
  if (elRpmLink) elRpmLink.href = linuxRpmUrl;
  const elRpmText = document.getElementById("linuxRpmBtnText");
  if (elRpmText) elRpmText.textContent = `Download .rpm (${archLabel})`;

  const macTarUrl = `https://github.com/Isu-Ismail/ctrlv/releases/latest/download/ctrlv-cli_darwin_${currentArch}.tar.gz`;
  const macLabel = (currentArch === "amd64") ? "Intel" : "Apple Silicon";
  const elCurl = document.getElementById("macCurlCmd");
  if (elCurl) elCurl.textContent = `curl -sSfL ${macTarUrl} | tar -xz && sudo mv ctrlv /usr/local/bin/`;
  const elTarLink = document.getElementById("macTarLink");
  if (elTarLink) elTarLink.href = macTarUrl;
  const elTarText = document.getElementById("macTarBtnText");
  if (elTarText) elTarText.textContent = `Download tar.gz (${macLabel})`;
}

osBtns.forEach(btn => {
  btn.addEventListener("click", () => {
    currentOs = btn.dataset.os;
    updateDlContent();
  });
});

archBtns.forEach(btn => {
  btn.addEventListener("click", () => {
    currentArch = btn.dataset.arch;
    updateDlContent();
  });
});

document.querySelectorAll(".btn-code-copy").forEach(btn => {
  btn.addEventListener("click", () => {
    const targetId = btn.dataset.target;
    const el = document.getElementById(targetId);
    if (!el) return;
    navigator.clipboard.writeText(el.textContent).then(() => {
      const originalText = btn.innerHTML;
      btn.innerHTML = `<i class="fa-solid fa-check"></i> Copied!`;
      btn.style.background = "#10b981";
      setTimeout(() => {
        btn.innerHTML = originalText;
        btn.style.background = "";
      }, 2000);
    });
  });
});

updateDlContent();

// Load and restore all cached screenshots, web text, and PC question text for current room
loadAndDisplayCachedData();

// NOTE: DO NOT auto-connect on load! Status stays Disconnected until user clicks Connect.
updateConnectionStatus(false);
