import React, { useState } from "react";
import styles from "./ChatBox.module.css";

export default function ChatBox({ onSendMessage, isLoading }) {
  const [input, setInput] = useState("");

  const handleSubmit = (e) => {
    e.preventDefault();
    if (input.trim()) {
      onSendMessage(input);
      setInput("");
    }
  };

  return (
    <form className={styles.container} onSubmit={handleSubmit}>
      <input
        type="text"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder="メッセージを入力..."
        disabled={isLoading}
        className={styles.input}
        autoFocus
      />
      <button
        type="submit"
        disabled={isLoading || !input.trim()}
        className={styles.button}
      >
        {isLoading ? "送信中..." : "送信"}
      </button>
    </form>
  );
}
