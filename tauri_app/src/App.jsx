import React, { useState, useCallback } from "react";
import ChatBox from "./components/ChatBox";
import MessageList from "./components/MessageList";
import DevLogger from "./components/DevLogger";
import { chatApi } from "./api/index.js";

function getStoredId(storage, key, prefix) {
  try {
    const existing = storage.getItem(key);
    if (existing) {
      return existing;
    }

    const generated = `${prefix}_${crypto.randomUUID()}`;
    storage.setItem(key, generated);
    return generated;
  } catch {
    return `${prefix}_${Date.now()}`;
  }
}

function App() {
  const [messages, setMessages] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [lastMetadata, setLastMetadata] = useState(null);
  const [userId] = useState(() => getStoredId(localStorage, "myn.userId", "user"));
  const [sessionId] = useState(() => getStoredId(sessionStorage, "myn.sessionId", "session"));
  const [error, setError] = useState(null);

  const handleSendMessage = useCallback(
    async (userMessage) => {
      // Add user message to chat
      const userMsg = {
        role: "user",
        content: userMessage,
        id: `msg_${Date.now()}`,
        timestamp_ms: Date.now(),
      };
      setMessages((prev) => [...prev, userMsg]);
      setIsLoading(true);
      setError(null);

      try {
        // Call Go backend
        const response = await chatApi.sendMessage(userId, userMessage, sessionId);

        if (response.error) {
          setError(response.error);
          setIsLoading(false);
          return;
        }

        // Add assistant response
        const assistantMsg = response.message;
        setMessages((prev) => [...prev, assistantMsg]);
        setLastMetadata(response.metadata);
      } catch (err) {
        const errMsg = err.message || "Unknown error";
        setError(`Failed to get response: ${errMsg}`);
        console.error("Chat error:", err);
      } finally {
        setIsLoading(false);
      }
    },
    [userId, sessionId]
  );

  return (
    <div className="app">
      <header className="header">
        <h1>🎙️ Myural Yukari</h1>
        <p className="subtitle">Local LLM Chat with Memory</p>
      </header>

      <main className="main">
        <MessageList messages={messages} isLoading={isLoading} metadata={lastMetadata} />

        {error && (
          <div className="error-banner">
            <span>⚠️ {error}</span>
            <button onClick={() => setError(null)}>✕</button>
          </div>
        )}

        <ChatBox onSendMessage={handleSendMessage} isLoading={isLoading} />
      </main>

      <DevLogger />
    </div>
  );
}

export default App;
