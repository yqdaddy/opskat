package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"sync/atomic"
	"time"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"
)

// Codex App Server JSON-RPC 2.0 适配器

// CodexAppServer 管理与 codex app-server 的通信
type CodexAppServer struct {
	cliPath string
	proc    *CLIProcess
	nextID  atomic.Int64

	// 响应等待
	pending   map[int64]chan codexJSONRPC
	pendingMu sync.Mutex

	// 通知事件分发
	notifyCh chan codexJSONRPC // 后台 reader 将通知事件发到这里
	ctx      context.Context
	cancel   context.CancelFunc

	// turn 级别互斥锁，防止多会话并发 turn 导致事件混串
	turnMu sync.Mutex

	// OnPermissionRequest 工具调用权限确认回调
	OnPermissionRequest func(req PermissionRequest) PermissionResponse

	// 工具确认响应 channel（会话内审批用）
	confirmCh chan PermissionResponse

	// approvedItems 跟踪已通过审批的 item ID，避免 item/started 重复创建 tool block
	approvedItems sync.Map
}

// NewCodexAppServer 创建 Codex App Server 客户端
func NewCodexAppServer() *CodexAppServer {
	return &CodexAppServer{
		pending:   make(map[int64]chan codexJSONRPC),
		notifyCh:  make(chan codexJSONRPC, 128),
		confirmCh: make(chan PermissionResponse, 1),
	}
}

// Start 启动 codex app-server 进程并完成初始化握手
func (s *CodexAppServer) Start(ctx context.Context, cliPath, workDir string, env map[string]string) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.cliPath = cliPath

	args := []string{"app-server"}
	proc, err := StartCLIProcess(s.ctx, cliPath, args, workDir, env)
	if err != nil {
		return err
	}
	s.proc = proc

	// 启动后台 stdout reader（必须在 sendRequest 之前启动）
	go s.readLoop()

	// 初始化握手
	if err := s.initialize(); err != nil {
		stderrStr := proc.Stderr()
		s.Stop()
		if stderrStr != "" {
			return fmt.Errorf("codex 初始化失败: %w\nstderr: %s", err, stderrStr)
		}
		return fmt.Errorf("codex 初始化失败: %w", err)
	}

	return nil
}

// readLoop 后台持续读取 stdout，分发到 pending 响应或 notifyCh
func (s *CodexAppServer) readLoop() {
	lines := s.proc.ReadLines(s.ctx)
	for line := range lines {
		var msg codexJSONRPC
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		// 有 ID 且无 method → 是对请求的响应
		if msg.ID != nil && msg.Method == "" {
			s.pendingMu.Lock()
			if ch, ok := s.pending[*msg.ID]; ok {
				ch <- msg
				delete(s.pending, *msg.ID)
			}
			s.pendingMu.Unlock()
			continue
		}

		// 否则是通知事件，发到 notifyCh
		select {
		case s.notifyCh <- msg:
		case <-s.ctx.Done():
			return
		}
	}
}

// initialize 发送 initialize 请求和 initialized 通知
func (s *CodexAppServer) initialize() error {
	version := getCLIVersion(s.cliPath)
	initParams := map[string]any{
		"clientInfo": map[string]any{
			"name":    "codex-cli",
			"version": version,
		},
	}
	_, err := s.sendRequest("initialize", initParams)
	if err != nil {
		return err
	}

	// 发送 initialized 通知（无 id）
	return s.sendNotification("initialized", nil)
}

// StartThread 开始新的对话线程，返回 thread ID
func (s *CodexAppServer) StartThread() (string, error) {
	result, err := s.sendRequest("thread/start", map[string]any{})
	if err != nil {
		return "", err
	}

	var resp struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("解析 thread/start 响应失败: %w", err)
	}
	return resp.Thread.ID, nil
}

// SendTurn 发送用户消息开始一个 turn
// threadID 为空时自动创建新 thread，返回实际使用的 threadID
func (s *CodexAppServer) SendTurn(ctx context.Context, threadID, text string, onEvent func(StreamEvent)) (string, error) {
	// turn 级别互斥：同一时间只能有一个 turn 在读取 notifyCh 事件，防止多会话事件混串
	s.turnMu.Lock()
	defer s.turnMu.Unlock()

	if threadID == "" {
		var err error
		threadID, err = s.StartThread()
		if err != nil {
			return "", err
		}
	}

	params := map[string]any{
		"threadId": threadID,
		"input":    []map[string]any{{"type": "text", "text": text}},
	}

	_, err := s.sendRequest("turn/start", params)
	if err != nil {
		return threadID, err
	}

	// 从 notifyCh 读取事件直到 turn 完成
	for {
		select {
		case msg := <-s.notifyCh:
			done := s.handleNotification(msg, onEvent)
			if done {
				return threadID, nil
			}
		case <-ctx.Done():
			return threadID, ctx.Err()
		}
	}
}

// handleNotification 处理 Codex 通知事件
func (s *CodexAppServer) handleNotification(msg codexJSONRPC, onEvent func(StreamEvent)) bool {
	method := msg.Method
	params := msg.Params
	logger.Default().Debug("codex notification", zap.String("method", method), zap.String("params", string(params)))
	switch method {
	// ── 文本流式输出 ──
	case "codex/event/agent_message_delta":
		var p struct {
			Msg struct {
				Delta string `json:"delta"`
			} `json:"msg"`
		}
		if err := json.Unmarshal(params, &p); err == nil && p.Msg.Delta != "" {
			onEvent(StreamEvent{Type: "content", Content: p.Msg.Delta})
		}

	// ── 命令执行 ──
	case "codex/event/exec_command_begin":
		var p struct {
			Msg struct {
				Command string `json:"command"`
			} `json:"msg"`
		}
		if err := json.Unmarshal(params, &p); err == nil && p.Msg.Command != "" {
			onEvent(StreamEvent{Type: "tool_start", ToolName: "Bash", ToolInput: p.Msg.Command})
		}

	case "codex/event/exec_command_end":
		var p struct {
			Msg struct {
				ExitCode int    `json:"exit_code"`
				Stdout   string `json:"stdout"`
				Stderr   string `json:"stderr"`
			} `json:"msg"`
		}
		if err := json.Unmarshal(params, &p); err == nil {
			result := p.Msg.Stdout
			if p.Msg.Stderr != "" {
				if result != "" {
					result += "\n"
				}
				result += p.Msg.Stderr
			}
			if p.Msg.ExitCode != 0 {
				result = fmt.Sprintf("exit code %d\n%s", p.Msg.ExitCode, result)
			}
			onEvent(StreamEvent{Type: "tool_result", ToolName: "Bash", Content: truncateOutput(result, 20)})
		}

	// ── item 事件（camelCase 类型名）──
	case "item/started":
		var p struct {
			Item codexItem `json:"item"`
		}
		if err := json.Unmarshal(params, &p); err == nil {
			s.handleItemStarted(&p.Item, onEvent)
		}

	case "item/completed":
		var p struct {
			Item codexItem `json:"item"`
		}
		if err := json.Unmarshal(params, &p); err == nil {
			s.handleItemCompleted(&p.Item, onEvent)
		}

	// ── 工具权限确认（只处理一种格式，避免弹两次）──
	case "item/tool/requestUserInput":
		s.handleUserInputRequest(msg.ID, params, onEvent)

	case "codex/event/request_user_input":
		logger.Default().Debug("codex request_user_input (codex/event)", zap.String("params", string(params)))

	// ── 本地命令执行审批 ──
	case "item/commandExecution/requestApproval":
		s.handleCommandApproval(msg.ID, params, onEvent)

	// ── turn 生命周期 ──
	case "turn/completed":
		return true

	case "turn/failed":
		var p struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(params, &p); err == nil {
			onEvent(StreamEvent{Type: "error", Error: p.Error})
		}
		return true

	// ── v1 item 事件 ──
	case "codex/event/item_started":
		var p struct {
			Item codexItem `json:"item"`
		}
		var p2 struct {
			Msg struct {
				Item codexItem `json:"item"`
			} `json:"msg"`
		}
		if err := json.Unmarshal(params, &p); err == nil && p.Item.Type != "" {
			s.handleItemStarted(&p.Item, onEvent)
		} else if err := json.Unmarshal(params, &p2); err == nil && p2.Msg.Item.Type != "" {
			s.handleItemStarted(&p2.Msg.Item, onEvent)
		}

	case "codex/event/item_completed":
		var p struct {
			Item codexItem `json:"item"`
		}
		var p2 struct {
			Msg struct {
				Item codexItem `json:"item"`
			} `json:"msg"`
		}
		if err := json.Unmarshal(params, &p); err == nil && p.Item.Type != "" {
			s.handleItemCompleted(&p.Item, onEvent)
		} else if err := json.Unmarshal(params, &p2); err == nil && p2.Msg.Item.Type != "" {
			s.handleItemCompleted(&p2.Msg.Item, onEvent)
		}

	// ── v1 文本流式输出（content_delta 格式）──
	case "codex/event/agent_message_content_delta":
		// 新版 Codex 可能通过此事件发送文本 delta，格式可能与 agent_message_delta 不同
		var p1 struct {
			Msg struct {
				Delta string `json:"delta"`
			} `json:"msg"`
		}
		var p2 struct {
			Delta string `json:"delta"`
		}
		var p3 struct {
			Msg struct {
				ContentDelta string `json:"content_delta"`
			} `json:"msg"`
		}
		if err := json.Unmarshal(params, &p1); err == nil && p1.Msg.Delta != "" {
			onEvent(StreamEvent{Type: "content", Content: p1.Msg.Delta})
		} else if err := json.Unmarshal(params, &p2); err == nil && p2.Delta != "" {
			onEvent(StreamEvent{Type: "content", Content: p2.Delta})
		} else if err := json.Unmarshal(params, &p3); err == nil && p3.Msg.ContentDelta != "" {
			onEvent(StreamEvent{Type: "content", Content: p3.Msg.ContentDelta})
		} else {
			logger.Default().Debug("codex agent_message_content_delta: unrecognized format", zap.String("params", string(params)))
		}

	// ── 静默忽略的事件 ──
	case "codex/event/agent_message",
		"codex/event/token_count",
		"codex/event/task_started",
		"codex/event/task_complete",
		"codex/event/user_message",
		"codex/event/mcp_startup_complete":
		// 静默忽略

	case "item/agentMessage/delta":
		// v2 流式文本，尝试多种可能的格式
		var p1 struct {
			Delta string `json:"delta"`
		}
		var p2 struct {
			Item struct {
				Delta string `json:"delta"`
			} `json:"item"`
		}
		if err := json.Unmarshal(params, &p1); err == nil && p1.Delta != "" {
			onEvent(StreamEvent{Type: "content", Content: p1.Delta})
		} else if err := json.Unmarshal(params, &p2); err == nil && p2.Item.Delta != "" {
			onEvent(StreamEvent{Type: "content", Content: p2.Item.Delta})
		}

	case
		"thread/started",
		"thread/status/changed",
		"thread/tokenUsage/updated",
		"account/rateLimits/updated",
		"turn/started",
		"configWarning",
		"serverRequest/resolved":
		// 忽略

	default:
		logger.Default().Debug("codex unhandled event", zap.String("method", method), zap.String("params", string(params)))
	}

	return false
}

func (s *CodexAppServer) handleItemStarted(item *codexItem, onEvent func(StreamEvent)) {
	switch item.Type {
	case "commandExecution":
		// 已通过审批的命令，confirm block 已存在，跳过重复创建 tool block
		if _, approved := s.approvedItems.LoadAndDelete(item.ID); approved {
			return
		}
		if item.Command != "" {
			onEvent(StreamEvent{Type: "tool_start", ToolName: "Bash", ToolInput: item.Command})
		}
	case "fileRead":
		if item.Path != "" {
			onEvent(StreamEvent{Type: "tool_start", ToolName: "Read", ToolInput: item.Path})
		}
	case "fileWrite":
		if item.Path != "" {
			onEvent(StreamEvent{Type: "tool_start", ToolName: "Write", ToolInput: item.Path})
		}
	case "mcpToolCall":
		name := item.Tool
		if name == "" {
			name = "MCP Tool"
		}
		input := string(item.Args)
		onEvent(StreamEvent{Type: "tool_start", ToolName: name, ToolInput: input})
	case "agentMessage", "userMessage", "reasoning":
		// 忽略
	default:
		name := item.Tool
		if name == "" {
			name = item.Type
		}
		input := string(item.Args)
		if input == "" {
			input = item.Command
		}
		if input == "" {
			input = item.Path
		}
		if name != "" {
			onEvent(StreamEvent{Type: "tool_start", ToolName: name, ToolInput: input})
		}
	}
}

func (s *CodexAppServer) handleItemCompleted(item *codexItem, onEvent func(StreamEvent)) {
	switch item.Type {
	case "commandExecution":
		result := item.Output
		if result == "" {
			result = item.AggregatedOutput
		}
		if item.ExitCode != nil && *item.ExitCode != 0 {
			result = fmt.Sprintf("exit code %d\n%s", *item.ExitCode, result)
		}
		onEvent(StreamEvent{Type: "tool_result", ToolName: "Bash", Content: result})
	case "fileRead":
		onEvent(StreamEvent{Type: "tool_result", ToolName: "Read", Content: truncateOutput(item.contentString(), 20)})
	case "fileWrite":
		onEvent(StreamEvent{Type: "tool_result", ToolName: "Write", Content: item.Path})
	case "mcpToolCall":
		name := item.Tool
		if name == "" {
			name = "MCP Tool"
		}
		result := item.mcpResultText()
		onEvent(StreamEvent{Type: "tool_result", ToolName: name, Content: truncateOutput(result, 20)})
	case "agentMessage", "userMessage", "reasoning":
		// 忽略，delta 已经发送过
	default:
		name := item.Tool
		if name == "" {
			name = item.Type
		}
		result := item.mcpResultText()
		if result == "" {
			result = item.Output
		}
		if result == "" {
			result = item.contentString()
		}
		if name != "" {
			onEvent(StreamEvent{Type: "tool_result", ToolName: name, Content: truncateOutput(result, 20)})
		}
	}
}

// RespondConfirm 发送工具确认响应（前端调用）
func (s *CodexAppServer) RespondConfirm(resp PermissionResponse) {
	select {
	case s.confirmCh <- resp:
	default:
	}
}

// handleUserInputRequest 处理 Codex 工具权限确认请求
// 工具调用自动同意，审批由 opsctl approval 机制负责
func (s *CodexAppServer) handleUserInputRequest(requestID *int64, params json.RawMessage, onEvent func(StreamEvent)) {
	var req struct {
		ThreadID  string                   `json:"threadId"`
		TurnID    string                   `json:"turnId"`
		ItemID    string                   `json:"itemId"`
		Questions []codexUserInputQuestion `json:"questions"`
	}
	if err := json.Unmarshal(params, &req); err != nil || len(req.Questions) == 0 {
		logger.Default().Warn("codex handleUserInputRequest: parse failed or no questions", zap.Error(err), zap.String("params", string(params)))
		return
	}

	q := req.Questions[0]
	logger.Default().Debug("codex handleUserInputRequest", zap.String("questionID", q.ID), zap.String("header", q.Header), zap.String("question", q.Question), zap.Any("options", q.Options), zap.Any("requestID", requestID))

	// 不发 tool_start 事件：item/started 已在 requestUserInput 之前触发，由 handleItemStarted 处理
	// 自动选择 "Approve this Session"（第二个选项）
	answer := "Allow"
	if len(q.Options) > 1 {
		answer = q.Options[1].Label // "Approve this Session"
	} else if len(q.Options) > 0 {
		answer = q.Options[0].Label // fallback: "Approve Once"
	}
	logger.Default().Debug("codex handleUserInputRequest: auto-approve", zap.String("answer", answer))

	// Codex 期望的 response 格式: {"answers": {"questionId": {"answers": ["Allow"]}}}
	responseResult := map[string]any{
		"answers": map[string]any{
			q.ID: map[string]any{
				"answers": []string{answer},
			},
		},
	}

	if requestID != nil {
		resultData, err := json.Marshal(responseResult)
		if err != nil {
			logger.Default().Warn("marshal codex response result", zap.Error(err))
		}
		replyMsg := codexJSONRPC{
			ID:     requestID,
			Result: resultData,
		}
		if err := s.proc.WriteJSON(replyMsg); err != nil {
			logger.Default().Error("codex resolveUserInput response write failed", zap.Error(err))
		}
	} else {
		if err := s.sendNotification("item/tool/resolveUserInput", map[string]any{
			"threadId": req.ThreadID,
			"turnId":   req.TurnID,
			"itemId":   req.ItemID,
			"answers": map[string]any{
				q.ID: map[string]any{
					"answers": []string{answer},
				},
			},
		}); err != nil {
			logger.Default().Error("codex sendNotification resolveUserInput failed", zap.Error(err))
		}
	}
}

// handleCommandApproval 处理 Codex 本地命令执行审批请求
func (s *CodexAppServer) handleCommandApproval(requestID *int64, params json.RawMessage, onEvent func(StreamEvent)) {
	var req struct {
		ThreadID string `json:"threadId"`
		TurnID   string `json:"turnId"`
		ItemID   string `json:"itemId"`
		Reason   string `json:"reason"`
		Command  string `json:"command"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		logger.Default().Warn("codex handleCommandApproval: parse failed", zap.Error(err), zap.String("params", string(params)))
		return
	}

	logger.Default().Debug("codex handleCommandApproval", zap.String("itemID", req.ItemID), zap.String("reason", req.Reason), zap.String("command", req.Command))

	// 记录已审批的 item ID，避免 item/started 重复创建 tool block
	s.approvedItems.Store(req.ItemID, struct{}{})

	// 生成 confirmID，通知前端在聊天内显示审批按钮
	confirmID := fmt.Sprintf("codex_%s_%d", req.ItemID, time.Now().UnixNano())
	onEvent(StreamEvent{
		Type:      "tool_confirm",
		ToolName:  "Bash",
		ToolInput: req.Command,
		Content:   req.Reason,
		ConfirmID: confirmID,
	})

	// 等待前端通过 confirmCh 响应（RespondCommandConfirm fallback 路径）
	decision := "accept"
	select {
	case resp := <-s.confirmCh:
		if resp.Behavior == "deny" {
			decision = "cancel"
			// 拒绝时不会有 item/started，清理记录
			s.approvedItems.Delete(req.ItemID)
		}
		// 通知前端更新 ToolBlock 状态
		resultContent := resp.Behavior
		if resultContent == "" {
			resultContent = "allow"
		}
		onEvent(StreamEvent{
			Type:      "tool_confirm_result",
			ConfirmID: confirmID,
			Content:   resultContent,
		})
	case <-s.ctx.Done():
		decision = "cancel"
		s.approvedItems.Delete(req.ItemID)
	}

	logger.Default().Debug("codex handleCommandApproval: decision", zap.String("decision", decision))

	// 回复审批决策
	response := map[string]any{
		"threadId": req.ThreadID,
		"turnId":   req.TurnID,
		"itemId":   req.ItemID,
		"decision": decision,
	}

	if requestID != nil {
		resultData, err := json.Marshal(response)
		if err != nil {
			logger.Default().Warn("marshal codex approval response", zap.Error(err))
		}
		replyMsg := codexJSONRPC{
			ID:     requestID,
			Result: resultData,
		}
		if err := s.proc.WriteJSON(replyMsg); err != nil {
			logger.Default().Error("codex handleCommandApproval response write failed", zap.Error(err))
		}
	} else {
		if err := s.sendNotification("item/commandExecution/resolveApproval", response); err != nil {
			logger.Default().Error("codex handleCommandApproval notification failed", zap.Error(err))
		}
	}
}

// Stop 停止 app-server 进程
func (s *CodexAppServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.proc != nil {
		s.proc.Stop()
	}
}
