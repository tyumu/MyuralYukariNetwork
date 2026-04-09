import React, { useState, useEffect, useRef } from "react";
import styles from "./DevLogger.module.css";
import { chatApi } from "../api/index.js";

export default function DevLogger() {
  const [logs, setLogs] = useState([]);
  const [isOpen, setIsOpen] = useState(true);
  const [services, setServices] = useState({});
  const endRef = useRef(null);

  useEffect(() => {
    // Pull status/logs every 5 seconds to keep dev noise low
    const interval = setInterval(async () => {
      try {
        const status = await chatApi.getStatus();
        setServices(status.services || {});
        setLogs(status.logs || []);
      } catch {
        // Silently fail for logs
      }
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "auto" });
  }, [logs]);

  return (
    <div className={`${styles.container} ${isOpen ? styles.open : styles.closed}`}>
      <button
        className={styles.toggle}
        onClick={() => setIsOpen(!isOpen)}
        title={isOpen ? "Close logs" : "Open logs"}
      >
        {isOpen ? "▼" : "▶"} Dev Logger
      </button>

      {isOpen && (
        <>
          <div className={styles.services}>
            {Object.entries(services).map(([name, status]) => (
              <span
                key={name}
                className={`${styles.service} ${styles[status]}`}
                title={`${name}: ${status}`}
              >
                {name}
              </span>
            ))}
          </div>

          <div className={styles.logs}>
            {logs.length === 0 && (
              <div className={styles.empty}>No logs yet</div>
            )}
            {logs.map((log, idx) => (
              <div key={idx} className={styles.log}>
                <span className={styles.time}>
                  {new Date().toLocaleTimeString()}
                </span>
                <span className={styles.text}>{log}</span>
              </div>
            ))}
            <div ref={endRef} />
          </div>
        </>
      )}
    </div>
  );
}
