package ai

import (
	"encoding/json"
	"fmt"
)

// Codex CLI exec --json JSONL 事件解析

// codexRawEvent codex exec --json 原始事件
type codexRawEvent struct {
	Type string          `json:"type"` // thread.started, turn.started, turn.completed, turn.failed, item.started, item.updated, item.completed, error
	Item *codexItem      `json:"item"`
	// turn.completed 时的 usage
	Usage *codexUsage    `json:"usage"`
	// thread.started 时的 thread_id
	ThreadID string      `json:"thread_id"`
	// error 时
	Error string         `json:"error"`
}

type codexItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`    // agent_message, command_execution, file_change, reasoning, etc.
	Text    string `json:"text"`    // agent_message 的文本
	Command string `json:"command"` // command_execution 的命令
	Status  string `json:"status"`  // in_progress, completed, etc.
}

type codexUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// CodexEventParser 解析 Codex exec --json 事件
type CodexEventParser struct {
	ThreadID string
}

// NewCodexEventParser 创建解析器
func NewCodexEventParser() *CodexEventParser {
	return &CodexEventParser{}
}

// ParseLine 解析一行 JSONL，返回 StreamEvent 和是否完成
func (p *CodexEventParser) ParseLine(line string) (events []StreamEvent, done bool) {
	if line == "" {
		return nil, false
	}

	var raw codexRawEvent
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, false
	}

	switch raw.Type {
	case "thread.started":
		if raw.ThreadID != "" {
			p.ThreadID = raw.ThreadID
		}

	case "item.started":
		if raw.Item != nil && raw.Item.Type == "command_execution" && raw.Item.Command != "" {
			return []StreamEvent{{
				Type:    "content",
				Content: fmt.Sprintf("\n🔧 `%s`\n", raw.Item.Command),
			}}, false
		}

	case "item.completed":
		if raw.Item != nil {
			switch raw.Item.Type {
			case "agent_message":
				if raw.Item.Text != "" {
					return []StreamEvent{{Type: "content", Content: raw.Item.Text}}, false
				}
			case "command_execution":
				// 命令完成，可显示状态
			}
		}

	case "turn.completed":
		return nil, true

	case "turn.failed":
		errMsg := "Codex turn 失败"
		if raw.Error != "" {
			errMsg = raw.Error
		}
		return []StreamEvent{{Type: "error", Error: errMsg}}, true

	case "error":
		return []StreamEvent{{Type: "error", Error: raw.Error}}, true
	}

	return nil, false
}
