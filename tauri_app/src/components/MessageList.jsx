import React, { useEffect, useRef } from "react";
import styles from "./MessageList.module.css";

export default function MessageList({ messages, isLoading, metadata }) {
  const endRef = useRef(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  return (
    <div className={styles.container}>
      {messages.length === 0 && (
        <div className={styles.empty}>
          <p>チャットを開始してください</p>
        </div>
      )}

      {messages.map((msg) => (
        <div key={msg.id} className={`${styles.message} ${styles[msg.role]}`}>
          <div className={styles.role}>
            {msg.role === "user" ? "👤 You" : "🤖 AI"}
          </div>
          <div className={styles.content}>{msg.content}</div>
          <div className={styles.timestamp}>
            {new Date(msg.timestamp_ms).toLocaleTimeString()}
          </div>
        </div>
      ))}

      {isLoading && (
        <div className={`${styles.message} ${styles.assistant}`}>
          <div className={styles.role}>🤖 AI</div>
          <div className={styles.loading}>
            <span></span><span></span><span></span>
          </div>
        </div>
      )}

      {metadata && (
        <div className={styles.metadata}>
          <details>
            <summary>📊 Response Metadata</summary>
            <pre>{JSON.stringify(metadata, null, 2)}</pre>
          </details>
        </div>
      )}

      <div ref={endRef} />
    </div>
  );
}
