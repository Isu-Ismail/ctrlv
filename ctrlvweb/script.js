import { initializeApp } from "https://www.gstatic.com/firebasejs/10.8.0/firebase-app.js";
import { getFirestore, doc, onSnapshot, setDoc, getDoc } from "https://www.gstatic.com/firebasejs/10.8.0/firebase-firestore.js";
import { getFirebaseConfig, getAIConfig } from "./config.js";

const firebaseConfig = getFirebaseConfig();

let app = null;
let db = null;

if (firebaseConfig) {
  try {
    app = initializeApp(firebaseConfig);
    db = getFirestore(app);
  } catch (err) {
    console.error("Firebase init error:", err);
  }
}

// UI Elements
const roomIdInput = document.getElementById("roomIdInput");
const btnConnect = document.getElementById("btnConnect");
const btnConnectText = document.getElementById("btnConnectText");
const connStatusBadge = document.getElementById("connStatusBadge");
const connStatusText = document.getElementById("connStatusText");
const screenshotImg = document.getElementById("screenshotImg");
const emptyState = document.getElementById("emptyState");
const textInput = document.getElementById("textInput");
const btnSendText = document.getElementById("btnSendText");
const btnSendTextLabel = document.getElementById("btnSendTextLabel");
const statusBadge = document.getElementById("statusBadge");
const imageModal = document.getElementById("imageModal");
const modalImg = document.getElementById("modalImg");
const btnCloseModal = document.getElementById("btnCloseModal");
const btnCopyRoom = document.getElementById("btnCopyRoom");
const btnToggleAutoDownload = document.getElementById("btnToggleAutoDownload");
const autoDlLabel = document.getElementById("autoDlLabel");
const btnDownloadImg = document.getElementById("btnDownloadImg");
const btnFetchScreenshot = document.getElementById("btnFetchScreenshot");
const fetchImgLabel = document.getElementById("fetchImgLabel");
const btnFetchPCText = document.getElementById("btnFetchPCText");
const fetchTextLabel = document.getElementById("fetchTextLabel");
const btnOpenHistory = document.getElementById("btnOpenHistory");

// Gemini & Multi-Provider AI Vision UI Elements
const btnToggleAutoSolve = document.getElementById("btnToggleAutoSolve");
const autoSolveLabel = document.getElementById("autoSolveLabel");
const btnSolveGemini = document.getElementById("btnSolveGemini");
const aiInstructionInput = document.getElementById("aiInstructionInput");
const aiSolverStatusBadge = document.getElementById("aiSolverStatusBadge");
const aiSolverStatusText = document.getElementById("aiSolverStatusText");

// Tabbed UI Elements
const tabBtnScreenshot = document.getElementById("tabBtnScreenshot");
const tabBtnPCText = document.getElementById("tabBtnPCText");
const screenshotViewer = document.getElementById("screenshotViewer");
const pcSentTextViewer = document.getElementById("pcSentTextViewer");
const pcSentTextDisplay = document.getElementById("pcSentTextDisplay");
const screenshotActions = document.getElementById("screenshotActions");
const pcTextActions = document.getElementById("pcTextActions");
const btnSolvePCSentText = document.getElementById("btnSolvePCSentText");

let activeTab = "screenshot"; // "screenshot" or "pctext"
let unsubscribe = null;
let isConnected = false;
let autoDownload = localStorage.getItem("ctrlv_auto_download") === "true";
let autoSolveEnabled = localStorage.getItem("ctrlv_auto_solve") !== "false"; // Default ON
let currentRoomId = localStorage.getItem("ctrlv_room_id") || "room-alpha-123";
let isSolvingAI = false;

if (roomIdInput) roomIdInput.value = currentRoomId;

// Populate initial prompt from AI config if saved
const savedAiConfig = getAIConfig();
if (aiInstructionInput && savedAiConfig.customPrompt) {
  aiInstructionInput.value = savedAiConfig.customPrompt;
}

// Cache variables to prevent laggy DOM re-renders
let cachedImagePath = null;
let cachedText = null;
let cachedFetchedStatus = null;
let cachedPCSentText = null;

// Tab Switching Logic
function switchTab(tab) {
  activeTab = tab;
  if (tab === "screenshot") {
    if (tabBtnScreenshot) tabBtnScreenshot.className = "tab-btn active";
    if (tabBtnPCText) tabBtnPCText.className = "tab-btn";
    if (screenshotViewer) screenshotViewer.style.display = "flex";
    if (pcSentTextViewer) pcSentTextViewer.style.display = "none";
    if (screenshotActions) screenshotActions.style.display = "flex";
    if (pcTextActions) pcTextActions.style.display = "none";
  } else {
    if (tabBtnScreenshot) tabBtnScreenshot.className = "tab-btn";
    if (tabBtnPCText) tabBtnPCText.className = "tab-btn active";
    if (screenshotViewer) screenshotViewer.style.display = "none";
    if (pcSentTextViewer) pcSentTextViewer.style.display = "block";
    if (screenshotActions) screenshotActions.style.display = "none";
    if (pcTextActions) pcTextActions.style.display = "flex";
  }
}

if (tabBtnScreenshot) tabBtnScreenshot.addEventListener("click", () => switchTab("screenshot"));
if (tabBtnPCText) tabBtnPCText.addEventListener("click", () => switchTab("pctext"));

// Persistent Local Storage Cache for Screenshot
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

function loadAndDisplayCachedScreenshot() {
  const cached = getCachedScreenshot();
  if (cached && cached.trim() !== "") {
    cachedImagePath = cached;
    screenshotImg.src = cached;
    screenshotImg.style.display = "block";
    emptyState.style.display = "none";
  }
}

// Initial UI Setup
updateAutoDownloadUI();
updateAutoSolveUI();

// Auto Download Toggle
function updateAutoDownloadUI() {
  if (!btnToggleAutoDownload) return;
  if (autoDownload) {
    btnToggleAutoDownload.className = "btn-tool active";
    autoDlLabel.textContent = "Auto-Save: ON";
  } else {
    btnToggleAutoDownload.className = "btn-tool";
    autoDlLabel.textContent = "Auto-Save: OFF";
  }
}

if (btnToggleAutoDownload) {
  btnToggleAutoDownload.addEventListener("click", () => {
    autoDownload = !autoDownload;
    localStorage.setItem("ctrlv_auto_download", autoDownload ? "true" : "false");
    updateAutoDownloadUI();
  });
}

// Auto Solve Toggle
function updateAutoSolveUI() {
  if (!btnToggleAutoSolve) return;
  if (autoSolveEnabled) {
    btnToggleAutoSolve.className = "btn-tool active";
    if (autoSolveLabel) autoSolveLabel.textContent = "Auto-Solve: ON";
  } else {
    btnToggleAutoSolve.className = "btn-tool";
    if (autoSolveLabel) autoSolveLabel.textContent = "Auto-Solve: OFF";
  }
}

if (btnToggleAutoSolve) {
  btnToggleAutoSolve.addEventListener("click", () => {
    autoSolveEnabled = !autoSolveEnabled;
    localStorage.setItem("ctrlv_auto_solve", autoSolveEnabled ? "true" : "false");
    updateAutoSolveUI();
  });
}

function updateAISolverStatus(state, msg) {
  if (!aiSolverStatusBadge || !aiSolverStatusText) return;
  if (state === "solving") {
    aiSolverStatusBadge.className = "status-pill pending";
    aiSolverStatusBadge.style.background = "#eff6ff";
    aiSolverStatusBadge.style.color = "#2563eb";
    aiSolverStatusBadge.style.borderColor = "#93c5fd";
    aiSolverStatusText.innerHTML = `<i class="fa-solid fa-spinner fa-spin"></i> ${msg || "AI Solving..."}`;
  } else if (state === "success") {
    aiSolverStatusBadge.className = "status-pill seen";
    aiSolverStatusText.innerHTML = `<i class="fa-solid fa-circle-check"></i> ${msg || "Solved & Pushed to PC!"}`;
  } else if (state === "error") {
    aiSolverStatusBadge.className = "status-pill offline";
    aiSolverStatusBadge.style.background = "#fef2f2";
    aiSolverStatusBadge.style.color = "#dc2626";
    aiSolverStatusBadge.style.borderColor = "#fca5a5";
    aiSolverStatusText.innerHTML = `<i class="fa-solid fa-triangle-exclamation"></i> ${msg || "AI Error"}`;
  } else {
    aiSolverStatusBadge.className = "status-pill pending";
    aiSolverStatusBadge.style.background = "";
    aiSolverStatusBadge.style.color = "";
    aiSolverStatusBadge.style.borderColor = "";
    aiSolverStatusText.innerHTML = `<i class="fa-solid fa-robot"></i> ${msg || "AI Ready"}`;
  }
}

// Function to call Multi-Provider Vision REST APIs (OpenRouter Auto, Groq Free, Google AI Studio)
async function solveImageWithGemini(b64ImageData, promptText) {
  if (isSolvingAI) return;

  const aiConfig = getAIConfig();
  if (!aiConfig || !aiConfig.apiKey) {
    updateAISolverStatus("error", "Configure AI Key in Config page first!");
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
  const provider = aiConfig.provider || "openrouter";
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
          "HTTP-Referer": "https://ctrlv.sync",
          "X-Title": "ctrlv Web AI Solver",
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
    await sendTextToFirestoreAuto(generatedText);
    updateAISolverStatus("success", "Solved Screenshot & Pushed to PC!");
  } catch (err) {
    console.error("AI Solver error:", err);
    updateAISolverStatus("error", err.message || "Failed to solve with AI");
  } finally {
    isSolvingAI = false;
  }
}

// Solve Question Text sent from PC (Ctrl+Shift+T)
async function solveTextWithGemini(questionText, promptText) {
  if (isSolvingAI) return;

  const aiConfig = getAIConfig();
  if (!aiConfig || !aiConfig.apiKey) {
    updateAISolverStatus("error", "Configure AI Key in Config page first!");
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
  const provider = aiConfig.provider || "openrouter";
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
          "HTTP-Referer": "https://ctrlv.sync",
          "X-Title": "ctrlv Web AI Solver",
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          model: model,
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
    await sendTextToFirestoreAuto(generatedText);
    updateAISolverStatus("success", "Text Solved & Pushed to PC!");
  } catch (err) {
    console.error("AI Text Solver error:", err);
    updateAISolverStatus("error", err.message || "AI Solving failed");
  } finally {
    isSolvingAI = false;
  }
}

// Function to extract clean code from markdown code fences
function parseCleanCodeOnly(rawText) {
  if (!rawText) return "";

  const codeBlockRegex = /```(?:[a-zA-Z0-9_+-]+)?\n([\s\S]*?)```/g;
  const matches = [...rawText.matchAll(codeBlockRegex)];

  if (matches.length > 0) {
    return matches.map(m => m[1].trim()).join("\n\n");
  }

  return rawText.trim();
}

async function sendTextToFirestoreAuto(cleanText) {
  if (!db || !currentRoomId) return;

  try {
    const formattedText = cleanText + "\n/---/";
    const roomRef = doc(db, "room", currentRoomId);
    await setDoc(roomRef, {
      uploaded_text: formattedText,
      fetched: false
    }, { merge: true });

    saveToHistory(currentRoomId, cleanText);
    console.log("Auto-pushed solution to Firestore!");
  } catch (err) {
    console.error("Auto push error:", err);
  }
}

if (btnSolveGemini) {
  btnSolveGemini.addEventListener("click", () => {
    const prompt = aiInstructionInput ? aiInstructionInput.value.trim() : "";
    if (activeTab === "pctext") {
      const question = pcSentTextDisplay ? pcSentTextDisplay.textContent.trim() : "";
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
    const question = pcSentTextDisplay ? pcSentTextDisplay.textContent.trim() : "";
    if (!question || question.startsWith("No question text received")) {
      updateAISolverStatus("error", "No question text available to solve!");
      return;
    }
    const prompt = aiInstructionInput ? aiInstructionInput.value.trim() : "";
    solveTextWithGemini(question, prompt);
  });
}

function saveToHistory(roomId, text) {
  if (!text || text.trim() === "") return;
  try {
    const key = `ctrlv_history_${roomId}`;
    let history = JSON.parse(localStorage.getItem(key) || "[]");
    
    history.unshift({
      id: Date.now(),
      text: text,
      time: new Date().toLocaleString()
    });
    
    if (history.length > 20) history = history.slice(0, 20);
    
    localStorage.setItem(key, JSON.stringify(history));
  } catch (e) {
    console.warn("Failed to save history:", e);
  }
}

function updateConnectionStatus(online) {
  isConnected = online;
  if (!connStatusBadge) return;
  if (online) {
    connStatusBadge.className = "conn-status-badge online";
    connStatusText.textContent = "Connected";
    btnConnect.className = "btn-create-room connected";
    btnConnectText.textContent = "Connected";
  } else {
    connStatusBadge.className = "conn-status-badge offline";
    connStatusText.textContent = "Disconnected";
    btnConnect.className = "btn-create-room";
    btnConnectText.textContent = "Connect";
  }
}

async function connectToRoom(roomId) {
  if (unsubscribe) unsubscribe();
  currentRoomId = roomId;
  localStorage.setItem("ctrlv_room_id", roomId);

  if (!firebaseConfig || !db) {
    updateConnectionStatus(false);
    return false;
  }

  cachedImagePath = null;
  cachedText = null;
  cachedFetchedStatus = null;
  cachedPCSentText = null;

  loadAndDisplayCachedScreenshot();

  const roomRef = doc(db, "room", roomId);

  try {
    const snap = await getDoc(roomRef);
    if (!snap.exists()) {
      await setDoc(roomRef, {
        fetched: false,
        image_path: "",
        uploaded_text: ""
      });
    }

    updateConnectionStatus(true);
  } catch (err) {
    console.warn("Initializing doc error:", err);
    updateConnectionStatus(false);
    return false;
  }

  unsubscribe = onSnapshot(roomRef, (snapshot) => {
    if (snapshot.exists()) {
      const data = snapshot.data();

      // 1. Screenshot Update (Triggers Screenshot AI ONLY, does NOT trigger Text AI)
      const newImg = data.image_path || "";
      if (newImg !== cachedImagePath) {
        const isFirstLoad = (cachedImagePath === null);
        cachedImagePath = newImg;
        if (newImg.trim() !== "") {
          screenshotImg.src = newImg;
          screenshotImg.style.display = "block";
          emptyState.style.display = "none";

          saveCachedScreenshot(newImg);

          if (autoDownload && !isFirstLoad) {
            downloadImage(newImg, `ctrlv-${currentRoomId}-${Date.now()}.png`);
          }

          if (autoSolveEnabled && !isFirstLoad) {
            switchTab("screenshot");
            const prompt = aiInstructionInput ? aiInstructionInput.value.trim() : "";
            solveImageWithGemini(newImg, prompt);
          }
        } else if (!getCachedScreenshot()) {
          screenshotImg.style.display = "none";
          emptyState.style.display = "flex";
        }
      }

      // 2. PC Question Text Update (Ctrl+Shift+T) (Triggers Text AI ONLY, does NOT trigger Screenshot AI)
      const pcSentText = data.pc_sent_text || "";
      if (pcSentText !== cachedPCSentText) {
        const isFirstLoad = (cachedPCSentText === null);
        cachedPCSentText = pcSentText;

        if (pcSentTextDisplay) {
          if (pcSentText.trim() !== "") {
            pcSentTextDisplay.textContent = pcSentText;
            pcSentTextDisplay.style.color = "var(--text-main)";
          } else {
            pcSentTextDisplay.textContent = "No question text received yet. Press Ctrl+Shift+T on PC to send clipboard text here.";
            pcSentTextDisplay.style.color = "var(--text-muted)";
          }
        }

        // Auto-solve text question ONLY if Auto-Solve is enabled & switch tab to PC text!
        if (autoSolveEnabled && !isFirstLoad && pcSentText.trim() !== "") {
          switchTab("pctext");
          const prompt = aiInstructionInput ? aiInstructionInput.value.trim() : "";
          solveTextWithGemini(pcSentText, prompt);
        }
      }

      // 3. Text Field Update (Strip /---/ signature for UI display)
      let rawText = data.uploaded_text || "";
      if (rawText.endsWith("\n/---/")) {
        rawText = rawText.slice(0, -6);
      } else if (rawText.endsWith("/---/")) {
        rawText = rawText.slice(0, -5);
      }

      if (rawText !== cachedText) {
        cachedText = rawText;
        if (document.activeElement !== textInput) {
          textInput.value = rawText;
        }
      }

      // 4. Status Pill Update
      const isFetched = data.fetched === true;
      if (isFetched !== cachedFetchedStatus) {
        cachedFetchedStatus = isFetched;
        if (isFetched) {
          statusBadge.className = "status-pill seen";
          statusBadge.innerHTML = `<i class="fa-solid fa-circle-check"></i> <span>Seen / Copied to PC</span>`;
        } else {
          statusBadge.className = "status-pill pending";
          statusBadge.innerHTML = `<i class="fa-solid fa-clock"></i> <span>Pending (Press Ctrl+Shift+F)</span>`;
        }
      }
    }
  }, (error) => {
    console.error("Firestore listener error:", error);
    updateConnectionStatus(false);
  });

  return true;
}

// Event Listeners
if (btnConnect) {
  btnConnect.addEventListener("click", () => {
    const rid = roomIdInput.value.trim();
    if (rid) {
      if (isConnected && rid === currentRoomId) {
        if (unsubscribe) unsubscribe();
        updateConnectionStatus(false);
      } else {
        connectToRoom(rid);
      }
    }
  });
}

if (roomIdInput) {
  roomIdInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") {
      const rid = roomIdInput.value.trim();
      if (rid) connectToRoom(rid);
    }
  });
}

if (btnSendText) {
  btnSendText.addEventListener("click", async () => {
    const text = textInput.value.trim();
    if (!text) return;
    if (!db || !currentRoomId) {
      alert("Please connect to a room first.");
      return;
    }

    try {
      const formattedText = text + "\n/---/";
      const roomRef = doc(db, "room", currentRoomId);
      await setDoc(roomRef, {
        uploaded_text: formattedText,
        fetched: false
      }, { merge: true });

      saveToHistory(currentRoomId, text);
      btnSendTextLabel.textContent = "Sent!";
      setTimeout(() => {
        btnSendTextLabel.textContent = "Send to PC";
      }, 2000);
    } catch (err) {
      console.error("Send text error:", err);
      alert("Failed to send text to PC: " + err.message);
    }
  });
}

if (btnCopyRoom) {
  btnCopyRoom.addEventListener("click", () => {
    const cmd = `ctrlv -r ${currentRoomId}`;
    navigator.clipboard.writeText(cmd).then(() => {
      const copyBtnText = document.getElementById("copyBtnText");
      if (copyBtnText) copyBtnText.textContent = "Copied!";
      setTimeout(() => {
        if (copyBtnText) copyBtnText.textContent = "Copy CLI Command";
      }, 2000);
    });
  });
}

// Fetch Screenshot button: DOES NOT INVOKE AI!
if (btnFetchScreenshot) {
  btnFetchScreenshot.addEventListener("click", async () => {
    if (!db || !currentRoomId) return;
    fetchImgLabel.textContent = "Fetching...";
    try {
      const roomRef = doc(db, "room", currentRoomId);
      const snap = await getDoc(roomRef);
      if (snap.exists() && snap.data().image_path) {
        const b64 = snap.data().image_path;
        screenshotImg.src = b64;
        screenshotImg.style.display = "block";
        emptyState.style.display = "none";
        saveCachedScreenshot(b64);
      }
    } catch (err) {
      console.error("Fetch screenshot error:", err);
    } finally {
      fetchImgLabel.textContent = "Fetch";
    }
  });
}

// Fetch PC Text button: DOES NOT INVOKE AI!
if (btnFetchPCText) {
  btnFetchPCText.addEventListener("click", async () => {
    if (!db || !currentRoomId) return;
    fetchTextLabel.textContent = "Fetching...";
    try {
      const roomRef = doc(db, "room", currentRoomId);
      const snap = await getDoc(roomRef);
      if (snap.exists() && snap.data().pc_sent_text) {
        const text = snap.data().pc_sent_text;
        if (pcSentTextDisplay) {
          pcSentTextDisplay.textContent = text;
          pcSentTextDisplay.style.color = "var(--text-main)";
        }
      }
    } catch (err) {
      console.error("Fetch PC text error:", err);
    } finally {
      fetchTextLabel.textContent = "Fetch Text";
    }
  });
}

if (btnDownloadImg) {
  btnDownloadImg.addEventListener("click", () => {
    if (screenshotImg.src && screenshotImg.style.display !== "none") {
      downloadImage(screenshotImg.src, `ctrlv-${currentRoomId}-${Date.now()}.png`);
    } else {
      alert("No screenshot available to download.");
    }
  });
}

function downloadImage(dataUrl, filename) {
  const a = document.createElement("a");
  a.href = dataUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
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

if (imageModal) {
  imageModal.addEventListener("click", (e) => {
    if (e.target === imageModal) {
      imageModal.style.display = "none";
    }
  });
}

if (btnOpenHistory) {
  btnOpenHistory.addEventListener("click", () => {
    window.location.href = `history.html?room=${encodeURIComponent(currentRoomId)}`;
  });
}

// Initial Auto-Connect on page load
if (firebaseConfig && db && currentRoomId) {
  connectToRoom(currentRoomId);
}
