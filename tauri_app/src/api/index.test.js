import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { chatApi } from "./index.js";

describe("chatApi", () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("sendMessage posts chat payload", async () => {
    fetch.mockResolvedValue(
      new Response(
        JSON.stringify({ user_id: "u1", message: { role: "assistant", content: "ok" } }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const result = await chatApi.sendMessage("u1", "hello", "s1");

    expect(fetch).toHaveBeenCalledTimes(1);
    const [url, options] = fetch.mock.calls[0];
    expect(url).toBe("http://localhost:8000/chat");
    expect(options.method).toBe("POST");
    expect(JSON.parse(options.body)).toEqual({
      user_id: "u1",
      message: "hello",
      session_id: "s1",
    });
    expect(result.user_id).toBe("u1");
  });

  it("getStatus throws on non-2xx", async () => {
    fetch.mockResolvedValue(new Response("upstream error", { status: 502 }));

    await expect(chatApi.getStatus()).rejects.toThrow("HTTP 502");
  });
});
