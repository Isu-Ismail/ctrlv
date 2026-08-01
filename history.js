import { initializeApp } from "https://www.gstatic.com/firebasejs/10.8.0/firebase-app.js";
import { getFirestore, doc, setDoc } from "https://www.gstatic.com/firebasejs/10.8.0/firebase-firestore.js";
import { getFirebaseConfig } from "./config.js";

const firebaseConfig = getFirebaseConfig();
const app = initializeApp(firebaseConfig);
const db = getFirestore(app);
 
// Get Room ID from URL parameters or localStorage
const urlParams = new URLSearchParams(window.location.search);
let currentRoomId = urlParams.get("room") || localStorage.getItem("ctrlv_room_id") || "room-alpha-123";

const activeRoomBadge = document.getElementById("activeRoomBadge");
const historySidebar = document.getElementById("historySidebar");
const historyDetailPanel = document.getElementById("historyDetailPanel");
const btnClearAllHistory = document.getElementById("btnClearAllHistory");

if (activeRoomBadge) activeRoomBadge.textContent = currentRoomId;

let selectedIndex = 0;

function getHistory() {
  try {
    const key = `ctrlv_history_${currentRoomId}`;
    const stored = localStorage.getItem(key);
    return stored ? JSON.parse(stored) : [];
  } catch (e) {
    return [];
  }
}

function saveHistoryList(history) {
  try {
    const key = `ctrlv_history_${currentRoomId}`;
    localStorage.setItem(key, JSON.stringify(history));
  } catch (e) {
    console.warn("Failed to save history list:", e);
  }
}

function renderHistoryPage() {
  const history = getHistory();

  if (history.length === 0) {
    historySidebar.innerHTML = `
      <div style="font-size:0.8rem; color:var(--text-dim); text-align:center; padding:1.5rem 0.5rem;">
        No history items
      </div>
    `;
    historyDetailPanel.innerHTML = `
      <div style="flex:1; display:flex; flex-direction:column; align-items:center; justify-content:center; padding:2rem; color:var(--text-muted); text-align:center;">
        <i class="fa-solid fa-folder-open" style="font-size:3rem; color:var(--text-dim); margin-bottom:1rem;"></i>
        <h3 style="font-weight:700; color:var(--text-main);">No History Available</h3>
        <p style="font-size:0.9rem; margin-top:0.5rem;">Texts you send to your PC will be saved here in your sidebar history.</p>
      </div>
    `;
    return;
  }

  // Ensure selectedIndex is within bounds
  if (selectedIndex >= history.length) selectedIndex = 0;

  // 1. Render Left Sidebar
  historySidebar.innerHTML = history.map((item, idx) => {
    const snippet = (item.text || '').trim().split('\n')[0] || 'Empty Text';
    return `
      <div class="history-sidebar-item ${idx === selectedIndex ? 'active' : ''}" data-idx="${idx}">
        <span class="history-sidebar-snippet">${escapeHtml(snippet)}</span>
        <span class="history-sidebar-time">
          <i class="fa-regular fa-clock"></i> ${escapeHtml(item.time || 'Recent')}
        </span>
      </div>
    `;
  }).join('');

  // 2. Render Right Detail Panel for Selected Item
  const selectedItem = history[selectedIndex];

  historyDetailPanel.innerHTML = `
    <div class="history-detail-header">
      <div style="font-size:0.88rem; font-weight:700; color:var(--text-muted); display:flex; align-items:center; gap:0.5rem;">
        <i class="fa-solid fa-clock-rotate-left" style="color:var(--accent-primary);"></i>
        <span>Sent: ${escapeHtml(selectedItem.time || 'Recent')}</span>
      </div>

      <div style="display:flex; align-items:center; gap:0.5rem;">
        <button class="btn-send" id="btnCopySelectedText" style="padding:0.45rem 1.1rem; font-size:0.82rem;">
          <i class="fa-regular fa-copy"></i> <span id="copyTextLabel">Copy Text</span>
        </button>
        <button class="btn-tool" id="btnSendSelectedToPC" style="background:#0284c7; color:white; border:none; padding:0.45rem 1rem; font-weight:700;">
          <i class="fa-solid fa-paper-plane"></i> Send to PC
        </button>
        <button class="btn-tool" id="btnDeleteSelectedText" style="color:#ef4444;" title="Delete this entry">
          <i class="fa-solid fa-trash-can"></i>
        </button>
      </div>
    </div>
    <textarea class="history-main-editor" id="historyMainEditor" readonly>${escapeHtml(selectedItem.text)}</textarea>
  `;

  // Attach Sidebar Click Listeners
  document.querySelectorAll(".history-sidebar-item").forEach(el => {
    el.addEventListener("click", () => {
      selectedIndex = parseInt(el.getAttribute("data-idx"));
      renderHistoryPage();
    });
  });

  // Attach Detail Action Listeners
  const btnCopySelectedText = document.getElementById("btnCopySelectedText");
  const copyTextLabel = document.getElementById("copyTextLabel");
  const btnSendSelectedToPC = document.getElementById("btnSendSelectedToPC");
  const btnDeleteSelectedText = document.getElementById("btnDeleteSelectedText");

  if (btnCopySelectedText) {
    btnCopySelectedText.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(selectedItem.text);
        copyTextLabel.textContent = "Copied!";
        btnCopySelectedText.style.background = "#059669";
        setTimeout(() => {
          copyTextLabel.textContent = "Copy Text";
          btnCopySelectedText.style.background = "var(--accent-cyan)";
        }, 1800);
      } catch (err) {
        console.error("Copy error:", err);
      }
    });
  }

  if (btnSendSelectedToPC) {
    btnSendSelectedToPC.addEventListener("click", async () => {
      try {
        btnSendSelectedToPC.innerHTML = `<i class="fa-solid fa-spinner fa-spin"></i> Sending...`;
        const roomRef = doc(db, "room", currentRoomId);
        await setDoc(roomRef, {
          uploaded_text: selectedItem.text,
          fetched: false
        }, { merge: true });

        btnSendSelectedToPC.innerHTML = `<i class="fa-solid fa-circle-check"></i> Sent to PC!`;
        btnSendSelectedToPC.style.background = "#059669";
        setTimeout(() => {
          btnSendSelectedToPC.innerHTML = `<i class="fa-solid fa-paper-plane"></i> Send to PC`;
          btnSendSelectedToPC.style.background = "#0284c7";
        }, 1800);
      } catch (err) {
        console.error("Send error:", err);
        btnSendSelectedToPC.innerHTML = `<i class="fa-solid fa-triangle-exclamation"></i> Error`;
      }
    });
  }

  if (btnDeleteSelectedText) {
    btnDeleteSelectedText.addEventListener("click", () => {
      let currentHistory = getHistory();
      currentHistory.splice(selectedIndex, 1);
      if (selectedIndex > 0) selectedIndex--;
      saveHistoryList(currentHistory);
      renderHistoryPage();
    });
  }
}

function escapeHtml(str) {
  if (!str) return "";
  return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

if (btnClearAllHistory) {
  btnClearAllHistory.addEventListener("click", () => {
    if (confirm("Are you sure you want to clear all history for this room?")) {
      saveHistoryList([]);
      selectedIndex = 0;
      renderHistoryPage();
    }
  });
}

// Initial Render
renderHistoryPage();
