// Helper module for reading/writing dynamic Firebase & Multi-Provider AI configurations
export function getFirebaseConfig() {
  try {
    const stored = localStorage.getItem("ctrlv_firebase_config");
    if (stored) {
      const parsed = JSON.parse(stored);
      const projId = parsed.projectId || parsed.project_id;
      if (projId && projId.trim() !== "") {
        return {
          projectId: projId.trim(),
          clientEmail: (parsed.clientEmail || parsed.client_email || "").trim(),
          authDomain: parsed.authDomain || `${projId.trim()}.firebaseapp.com`
        };
      }
    }
  } catch (e) {
    console.warn("Failed to parse custom firebase config:", e);
  }

  // Returns null if user has not set Project ID & Client Email
  return null;
}

export function saveFirebaseConfig(projId, clientEmail) {
  const config = {
    projectId: (projId || "").trim(),
    clientEmail: (clientEmail || "").trim()
  };
  localStorage.setItem("ctrlv_firebase_config", JSON.stringify(config));
}

export function clearFirebaseConfig() {
  localStorage.removeItem("ctrlv_firebase_config");
}

// AI Multi-Provider Config Helpers
export function getAIConfig() {
  try {
    const stored = localStorage.getItem("ctrlv_ai_config");
    if (stored) {
      const parsed = JSON.parse(stored);
      return {
        provider: parsed.provider || "openrouter",
        apiKey: (parsed.apiKey || "").trim(),
        model: parsed.model || "openrouter/auto",
        codeOnly: parsed.codeOnly !== false,
        customPrompt: parsed.customPrompt || "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting."
      };
    }
  } catch (e) {
    console.warn("Failed to parse AI config:", e);
  }

  return {
    provider: "openrouter",
    apiKey: "",
    model: "openrouter/auto",
    codeOnly: true,
    customPrompt: "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting."
  };
}

export function saveAIConfig(provider, apiKey, model, codeOnly, customPrompt) {
  const config = {
    provider: provider || "openrouter",
    apiKey: (apiKey || "").trim(),
    model: model || "openrouter/auto",
    codeOnly: codeOnly !== false,
    customPrompt: customPrompt || "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting."
  };
  localStorage.setItem("ctrlv_ai_config", JSON.stringify(config));
}

export function clearAIConfig() {
  localStorage.removeItem("ctrlv_ai_config");
}
