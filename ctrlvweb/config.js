// Helper module for reading/writing Multi-Provider AI configurations
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
