// Package shared は Go、Python、フロントエンド間で共有する共通データ構造を定義します。
package shared

// ChatMessage は会話内の 1 件のチャットメッセージを表します。
type ChatMessage struct {
	Role        string `json:"role"`         // 'user' または 'assistant'
	Content     string `json:"content"`      // メッセージ本文
	ID          string `json:"id"`           // 一意なメッセージ ID
	TimestampMs int64  `json:"timestamp_ms"` // Unix ミリ秒
}

// ChatRequest はチャット操作のリクエスト本文です。
type ChatRequest struct {
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"` // マルチターン文脈用の任意項目
}

// ChatResponse はチャット API のレスポンスです。
type ChatResponse struct {
	UserID    string                 `json:"user_id"`
	Message   ChatMessage            `json:"message"`
	SessionID string                 `json:"session_id"`
	Metadata  map[string]interface{} `json:"metadata"` // 開発用ログ: メモリ統計、トークン数など
	Error     string                 `json:"error,omitempty"`
}

// MemorizeRequest は Python サイドカー経由でメモリ保存を依頼します。
type MemorizeRequest struct {
	UserID        string `json:"user_id"`
	Text          string `json:"text"`
	AssistantText string `json:"assistant_text,omitempty"`
	MemoryType    string `json:"memory_type"` // 'event', 'fact' など
	Category      string `json:"category"`    // 'conversation', 'user_profile' など
}

// MemorizeResponse はメモリ保存処理のレスポンスです。
type MemorizeResponse struct {
	UserID     string `json:"user_id"`
	ItemID     string `json:"item_id"` // 保存されたメモリアイテム ID
	Category   string `json:"category"`
	Success    bool   `json:"success"`
	Saved      bool   `json:"saved"`                 // true の場合のみ実保存された
	SkipReason string `json:"skip_reason,omitempty"` // 未保存時の理由
	Error      string `json:"error,omitempty"`
}

// RecallRequest は Python サイドカー経由でメモリ取得を依頼します。
type RecallRequest struct {
	UserID string                 `json:"user_id"`
	Query  string                 `json:"query"`
	TopK   int                    `json:"top_k"`
	Where  map[string]interface{} `json:"where,omitempty"` // 追加フィルタ
	Method string                 `json:"method"`          // 'rag' または 'llm'
}

// RecallItem は取得されたメモリ 1 件です。
type RecallItem struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Category string  `json:"category"`
	Salience float64 `json:"salience"` // 関連度スコア
}

// RecallResponse はメモリ取得処理のレスポンスです。
type RecallResponse struct {
	UserID  string       `json:"user_id"`
	Items   []RecallItem `json:"items"`
	Success bool         `json:"success"`
	Error   string       `json:"error,omitempty"`
}

// OpenAIChatMessage は llama.cpp の OpenAI 互換 chat API のメッセージです。
type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIChatCompletionRequest は llama.cpp の OpenAI 互換 chat completion リクエストです。
type OpenAIChatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenAIChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

// OpenAIChatCompletionChoice は chat completion の 1 候補です。
type OpenAIChatCompletionChoice struct {
	Message OpenAIChatMessage `json:"message"`
}

// OpenAIChatCompletionResponse は llama.cpp の OpenAI 互換 chat completion レスポンスです。
type OpenAIChatCompletionResponse struct {
	Choices []OpenAIChatCompletionChoice `json:"choices"`
}

// StatusResponse はヘルスチェックとログ情報を提供します。
type StatusResponse struct {
	Status      string            `json:"status"`   // 'healthy' | 'degraded' | 'error'
	Services    map[string]string `json:"services"` // サービス名 → 状態
	TimestampMs int64             `json:"timestamp_ms"`
	Logs        []string          `json:"logs,omitempty"` // デバッグ用の開発ログ
}
