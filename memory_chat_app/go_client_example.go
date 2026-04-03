package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MemorizeRequest struct {
	UserID     string `json:"user_id"`
	Text       string `json:"text"`
	MemoryType string `json:"memory_type"`
	Category   string `json:"category"`
}

type RecallRequest struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
	TopK   int    `json:"top_k"`
}

func postJSON(baseURL, path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(baseURL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(data))
	}
	return data, nil
}

func main() {
	baseURL := "http://127.0.0.1:8090"

	memorizeRes, err := postJSON(baseURL, "/memorize", MemorizeRequest{
		UserID:     "demo_user",
		Text:       "User likes melon soda.",
		MemoryType: "event",
		Category:   "conversation",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("/memorize:", string(memorizeRes))

	recallRes, err := postJSON(baseURL, "/recall", RecallRequest{
		UserID: "demo_user",
		Query:  "What does the user like?",
		TopK:   3,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("/recall:", string(recallRes))
}
