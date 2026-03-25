package ai

import (
	"encoding/json"
	"fmt"
	"time"
)

// Codex App Server JSON-RPC 2.0 协议类型与编解码

// codexJSONRPC JSON-RPC 2.0 消息
type codexJSONRPC struct {
	Method string          `json:"method,omitempty"`
	ID     *int64          `json:"id,omitempty"`     // 请求时设置，通知时不设置
	Params json.RawMessage `json:"params,omitempty"` // 请求参数
	Result json.RawMessage `json:"result,omitempty"` // 响应结果
	Error  *codexRPCError  `json:"error,omitempty"`  // 错误
}

type codexRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// codexItem Codex item 通用结构（camelCase 类型名）
type codexItem struct {
	Type             string          `json:"type"`
	ID               string          `json:"id"`
	Command          string          `json:"command"`          // commandExecution
	Path             string          `json:"path"`             // fileRead / fileWrite
	Output           string          `json:"output"`           // commandExecution completed
	AggregatedOutput string          `json:"aggregatedOutput"` // commandExecution completed (Codex 格式)
	Content          json.RawMessage `json:"content"`          // 可能是 string 或 array
	ExitCode         *int            `json:"exitCode"`         // commandExecution completed
	Text             string          `json:"text"`             // agentMessage completed
	Tool             string          `json:"tool"`             // mcpToolCall: 工具名
	Server           string          `json:"server"`           // mcpToolCall: MCP server 名
	Args             json.RawMessage `json:"arguments"`        // mcpToolCall: 参数（JSON）
	Result           json.RawMessage `json:"result"`           // mcpToolCall: 结果（嵌套结构）
}

// contentString 安全提取 content 字段为字符串
func (item *codexItem) contentString() string {
	if item.Content == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(item.Content, &s); err == nil {
		return s
	}
	return ""
}

// mcpResultText 提取 MCP 工具调用的结果文本
// 结果结构: {"content":[{"type":"text","text":"..."}]}
func (item *codexItem) mcpResultText() string {
	if item.Result == nil {
		return ""
	}
	var r struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(item.Result, &r); err != nil {
		return ""
	}
	var texts []string
	for _, c := range r.Content {
		if c.Text != "" {
			texts = append(texts, c.Text)
		}
	}
	if len(texts) == 1 {
		return texts[0]
	}
	result := ""
	for _, t := range texts {
		if result != "" {
			result += "\n"
		}
		result += t
	}
	return result
}

// codexUserInputQuestion Codex 用户输入请求的问题结构
type codexUserInputQuestion struct {
	ID       string `json:"id"`
	Header   string `json:"header"`
	Question string `json:"question"`
	Options  []struct {
		Label       string `json:"label"`
		Description string `json:"description"`
	} `json:"options"`
}

// sendRequest 发送 JSON-RPC 请求并等待响应
func (s *CodexAppServer) sendRequest(method string, params any) (json.RawMessage, error) {
	id := s.nextID.Add(1)
	paramsData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	ch := make(chan codexJSONRPC, 1)
	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()

	msg := codexJSONRPC{
		Method: method,
		ID:     &id,
		Params: paramsData,
	}
	if err := s.proc.WriteJSON(msg); err != nil {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("codex RPC 错误: %s", resp.Error.Message)
		}
		return resp.Result, nil
	case <-time.After(30 * time.Second):
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		stderrStr := s.proc.Stderr()
		if stderrStr != "" {
			return nil, fmt.Errorf("codex 请求超时 (%s)\nstderr: %s", method, stderrStr)
		}
		return nil, fmt.Errorf("codex 请求超时: %s", method)
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

// sendNotification 发送 JSON-RPC 通知（无 id）
func (s *CodexAppServer) sendNotification(method string, params any) error {
	var paramsData json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		paramsData = data
	}

	msg := codexJSONRPC{
		Method: method,
		Params: paramsData,
	}
	return s.proc.WriteJSON(msg)
}

// truncateOutput 截断长输出
func truncateOutput(s string, maxLines int) string {
	lines := 0
	for i, ch := range s {
		if ch == '\n' {
			lines++
			if lines >= maxLines {
				remaining := 0
				for _, c := range s[i+1:] {
					if c == '\n' {
						remaining++
					}
				}
				if remaining > 0 {
					return s[:i] + fmt.Sprintf("\n... (%d more lines)", remaining)
				}
				return s
			}
		}
	}
	return s
}
