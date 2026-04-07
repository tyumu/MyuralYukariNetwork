/**
 * API client for communicating with Go backend
 */

const API_BASE = import.meta.env.VITE_API_BASE_URL || "http://localhost:8000";

export const chatApi = {
  async sendMessage(userId, message, sessionId) {
    try {
      const response = await fetch(`${API_BASE}/chat`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: userId,
          message: message,
          session_id: sessionId || "",
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error("Chat API error:", error);
      throw error;
    }
  },

  async getStatus() {
    try {
      const response = await fetch(`${API_BASE}/status`, {
        method: "GET",
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error("Status API error:", error);
      throw error;
    }
  },
};
