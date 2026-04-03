import { invoke } from "@tauri-apps/api/core";

const greetBtn = document.getElementById("greet-btn");
const greetOut = document.getElementById("greet-output");
const nameInput = document.getElementById("name-input");

const goHealthBtn = document.getElementById("go-health-btn");
const goOut = document.getElementById("go-output");
const goBaseUrlInput = document.getElementById("go-base-url");

greetBtn?.addEventListener("click", async () => {
  try {
    const name = nameInput?.value?.trim() || "world";
    const msg = await invoke("greet", { name });
    greetOut.textContent = String(msg);
  } catch (error) {
    greetOut.textContent = `Tauri command error: ${String(error)}`;
  }
});

goHealthBtn?.addEventListener("click", async () => {
  try {
    const baseUrl = (goBaseUrlInput?.value || "").trim().replace(/\/$/, "");
    if (!baseUrl) {
      goOut.textContent = "Go API base URL is empty.";
      return;
    }
    const res = await fetch(`${baseUrl}/health`);
    const text = await res.text();
    goOut.textContent = `status=${res.status} body=${text}`;
  } catch (error) {
    goOut.textContent = `Fetch error: ${String(error)}`;
  }
});
