package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/opskat/opskat/internal/ai"
	"github.com/opskat/opskat/internal/bootstrap"
	"github.com/opskat/opskat/internal/model/entity/conversation_entity"
	"github.com/opskat/opskat/internal/service/conversation_svc"
	"github.com/opskat/opskat/internal/service/credential_svc"

	"github.com/cago-frame/cago/pkg/logger"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

// AISettingInfo AI 配置信息（返回给前端，API Key 已脱敏）
type AISettingInfo struct {
	ProviderType string `json:"providerType"`
	APIBase      string `json:"apiBase"`
	MaskedAPIKey string `json:"maskedApiKey"`
	Model        string `json:"model"`
	Configured   bool   `json:"configured"`
}

// SaveAISetting 保存 AI 配置（加密 API Key）并激活 provider
func (a *App) SaveAISetting(providerType, apiBase, apiKey, model string) error {
	cfg := bootstrap.GetConfig()
	cfg.AIProviderType = providerType
	cfg.AIAPIBase = apiBase
	cfg.AIModel = model

	actualKey := apiKey
	if apiKey != "" {
		encrypted, err := credential_svc.Default().Encrypt(apiKey)
		if err != nil {
			return fmt.Errorf("加密 API Key 失败: %w", err)
		}
		cfg.AIAPIKey = encrypted
	} else if cfg.AIAPIKey != "" {
		// 未提供新 key，解密已有 key 用于激活
		decrypted, err := credential_svc.Default().Decrypt(cfg.AIAPIKey)
		if err != nil {
			return fmt.Errorf("解密 API Key 失败: %w", err)
		}
		actualKey = decrypted
	}

	if err := bootstrap.SaveConfig(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	return a.SetAIProvider(providerType, apiBase, actualKey, model)
}

// LoadAISetting 加载已保存的 AI 配置并激活 provider
func (a *App) LoadAISetting() (*AISettingInfo, error) {
	cfg := bootstrap.GetConfig()
	if cfg == nil || cfg.AIProviderType == "" {
		return &AISettingInfo{Configured: false}, nil
	}

	info := &AISettingInfo{
		ProviderType: cfg.AIProviderType,
		APIBase:      cfg.AIAPIBase,
		Model:        cfg.AIModel,
		Configured:   true,
	}

	var apiKey string
	if cfg.AIAPIKey != "" {
		decrypted, err := credential_svc.Default().Decrypt(cfg.AIAPIKey)
		if err != nil {
			return nil, fmt.Errorf("解密 API Key 失败: %w", err)
		}
		apiKey = decrypted
		info.MaskedAPIKey = maskAPIKey(decrypted)
	}

	if err := a.SetAIProvider(cfg.AIProviderType, cfg.AIAPIBase, apiKey, cfg.AIModel); err != nil {
		return nil, err
	}

	return info, nil
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// SetAIProvider 设置 AI provider 并创建 agent
func (a *App) SetAIProvider(providerType, apiBase, apiKey, model string) error {
	a.aiProviderType = providerType
	a.aiModel = model

	// 创建共用的命令权限检查器
	checker := ai.NewCommandPolicyChecker(a.makeCommandConfirmFunc())
	checker.SetGrantRequestFunc(a.makeGrantRequestFunc())
	var provider ai.Provider
	switch providerType {
	case "openai":
		provider = ai.NewOpenAIProvider("OpenAI Compatible", apiBase, apiKey, model)
	case "local_cli":
		// apiBase 作为 CLI 路径，model 作为 CLI 类型
		cliProvider := ai.NewLocalCLIProvider("Local CLI", apiBase, model)
		// 注入权限确认回调：转发到前端，等待用户响应
		cliProvider.OnPermissionRequest = func(req ai.PermissionRequest) ai.PermissionResponse {
			wailsRuntime.EventsEmit(a.ctx, "ai:permission", req)
			select {
			case resp := <-a.permissionChan:
				return resp
			case <-a.shutdownCh:
				return ai.PermissionResponse{Behavior: "deny"}
			}
		}
		// 设置工作目录
		if a.currentConversationID > 0 {
			conv, err := conversation_svc.Conversation().Get(a.langCtx(), a.currentConversationID)
			if err == nil && conv.WorkDir != "" {
				cliProvider.SetWorkDir(conv.WorkDir)
			}
		}
		a.aiProvider = cliProvider
		a.aiAgent = ai.NewAgent(cliProvider, nil, checker)
		return nil
	default:
		provider = ai.NewOpenAIProvider(providerType, apiBase, apiKey, model)
	}
	a.aiAgent = ai.NewAgent(provider, ai.NewAuditingExecutor(ai.NewDefaultToolExecutor(), ai.NewDefaultAuditWriter()), checker)
	return nil
}

// --- AI 操作 ---

// ConversationDisplayMessage 返回给前端的会话消息（用于恢复显示）
type ConversationDisplayMessage struct {
	Role    string                             `json:"role"`
	Content string                             `json:"content"`
	Blocks  []conversation_entity.ContentBlock `json:"blocks"`
}

// CreateConversation 创建新会话
func (a *App) CreateConversation() (*conversation_entity.Conversation, error) {
	if a.aiAgent == nil {
		return nil, fmt.Errorf("请先配置 AI Provider")
	}

	ctx := a.langCtx()
	conv := &conversation_entity.Conversation{
		Title:        "新对话",
		ProviderType: a.aiProviderType,
		Model:        a.aiModel,
	}

	// 本地 CLI 模式创建工作目录
	if a.aiProviderType == "local_cli" {
		workDir := filepath.Join(bootstrap.AppDataDir(), "workspaces", fmt.Sprintf("conv-%d", time.Now().UnixMilli()))
		conv.WorkDir = workDir
	}

	if err := conversation_svc.Conversation().Create(ctx, conv); err != nil {
		return nil, err
	}

	// 如果有工作目录，更新路径为带 ID 的稳定路径
	if conv.WorkDir != "" {
		stableDir := filepath.Join(bootstrap.AppDataDir(), "workspaces", fmt.Sprintf("%d", conv.ID))
		if err := os.Rename(conv.WorkDir, stableDir); err == nil {
			conv.WorkDir = stableDir
			if err := conversation_svc.Conversation().Update(ctx, conv); err != nil {
				logger.Default().Error("update conversation work dir", zap.Error(err))
			}
		}
	}

	// 切换到新会话
	a.switchToConversation(conv)

	return conv, nil
}

// ListConversations 获取会话列表
func (a *App) ListConversations() ([]*conversation_entity.Conversation, error) {
	return conversation_svc.Conversation().List(a.langCtx())
}

// SwitchConversation 切换到指定会话，返回显示消息
func (a *App) SwitchConversation(id int64) ([]ConversationDisplayMessage, error) {
	ctx := a.langCtx()
	conv, err := conversation_svc.Conversation().Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("会话不存在: %w", err)
	}

	a.switchToConversation(conv)

	// 加载消息用于前端显示
	msgs, err := conversation_svc.Conversation().LoadMessages(ctx, id)
	if err != nil {
		return nil, err
	}

	var displayMsgs []ConversationDisplayMessage
	for _, msg := range msgs {
		blocks, err := msg.GetBlocks()
		if err != nil {
			logger.Default().Warn("get message blocks", zap.Error(err))
		}
		displayMsgs = append(displayMsgs, ConversationDisplayMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Blocks:  blocks,
		})
	}
	return displayMsgs, nil
}

// switchToConversation 内部切换会话逻辑
func (a *App) switchToConversation(conv *conversation_entity.Conversation) {
	a.currentConversationID = conv.ID

	if p, ok := a.aiProvider.(*ai.LocalCLIProvider); ok {
		// 恢复 CLI session
		info, err := conv.GetSessionInfo()
		if err == nil && info.SessionID != "" {
			p.SetClaudeSession(conv.ID, info.SessionID)
		} else {
			p.SetClaudeSession(conv.ID, "")
		}

		// 切换工作目录
		if conv.WorkDir != "" {
			p.SetWorkDir(conv.WorkDir)
		}
	}
}

// DeleteConversation 删除会话
func (a *App) DeleteConversation(id int64) error {
	err := conversation_svc.Conversation().Delete(a.langCtx(), id)
	if err != nil {
		return err
	}
	// 清理 Codex thread 映射
	if p, ok := a.aiProvider.(*ai.LocalCLIProvider); ok {
		p.ResetCodexThread(id)
	}
	// 如果删的是当前会话，清空当前会话ID
	if a.currentConversationID == id {
		a.currentConversationID = 0
	}
	return nil
}

// SendAIMessage 发送 AI 消息，通过 Wails Events 流式返回
// convID 指定目标会话，支持多会话并发发送
func (a *App) SendAIMessage(convID int64, messages []ai.Message) error {
	if a.aiAgent == nil {
		return fmt.Errorf("请先配置 AI Provider")
	}

	ctx := a.langCtx()

	// 自动创建会话（首次发消息时）
	if convID == 0 {
		conv, err := a.CreateConversation()
		if err != nil {
			return fmt.Errorf("创建会话失败: %w", err)
		}
		convID = conv.ID
	}

	// 更新会话标题（如果仍是默认标题"新对话"）
	if conv, err := conversation_svc.Conversation().Get(ctx, convID); err == nil && conv.Title == "新对话" {
		for _, msg := range messages {
			if msg.Role == ai.RoleUser {
				title := string(msg.Content)
				if len([]rune(title)) > 50 {
					title = string([]rune(title)[:50])
				}
				conv.Title = title
				if err := conversation_svc.Conversation().Update(ctx, conv); err != nil {
					logger.Default().Error("update conversation title", zap.Error(err))
				}
				break
			}
		}
	}

	eventName := fmt.Sprintf("ai:event:%d", convID)

	// 添加系统提示
	fullMessages := make([]ai.Message, 0, 1+len(messages))
	fullMessages = append(fullMessages, ai.Message{
		Role:    ai.RoleSystem,
		Content: "You are the OpsKat AI assistant, helping users manage IT assets. You can list assets, view details, add assets, and run commands on SSH servers. Respond in the same language the user uses.",
	})
	fullMessages = append(fullMessages, messages...)

	go func() {
		// 注入审计上下文
		chatCtx := ai.WithAuditSource(a.ctx, "ai")
		chatCtx = ai.WithConversationID(chatCtx, convID)
		chatCtx = ai.WithSessionID(chatCtx, fmt.Sprintf("conv_%d", convID))
		// 注入 SSH 连接池，供 Redis/Database SSH 隧道使用
		if a.sshPool != nil {
			chatCtx = ai.WithSSHPool(chatCtx, a.sshPool)
		}

		err := a.aiAgent.Chat(chatCtx, fullMessages, func(event ai.StreamEvent) {
			wailsRuntime.EventsEmit(a.ctx, eventName, event)
		})
		if err != nil {
			wailsRuntime.EventsEmit(a.ctx, eventName, ai.StreamEvent{
				Type:  "error",
				Error: err.Error(),
			})
		}

		// 消息完成后持久化
		a.persistConversationState(convID, messages)
	}()

	return nil
}

// persistConversationState 持久化会话状态（消息+session）
func (a *App) persistConversationState(convID int64, messages []ai.Message) {
	ctx := a.langCtx()

	// 保存 local CLI session ID
	if p, ok := a.aiProvider.(*ai.LocalCLIProvider); ok {
		conv, err := conversation_svc.Conversation().Get(ctx, convID)
		if err == nil {
			sessionID := p.GetClaudeSession(convID)
			if err := conv.SetSessionInfo(&conversation_entity.SessionInfo{
				SessionID: sessionID,
			}); err != nil {
				logger.Default().Error("set conversation session info", zap.Error(err))
			}
			conv.Updatetime = time.Now().Unix()
			if err := conversation_svc.Conversation().Update(ctx, conv); err != nil {
				logger.Default().Error("update conversation session info", zap.Error(err))
			}
		}
	}
}

// SaveConversationMessages 前端调用，保存显示消息到数据库
// convID 指定目标会话，支持多会话独立保存
func (a *App) SaveConversationMessages(convID int64, displayMsgs []ConversationDisplayMessage) error {
	if convID == 0 {
		return nil
	}
	ctx := a.langCtx()
	var msgs []*conversation_entity.Message
	for i, dm := range displayMsgs {
		msg := &conversation_entity.Message{
			ConversationID: convID,
			Role:           dm.Role,
			Content:        dm.Content,
			SortOrder:      i,
			Createtime:     time.Now().Unix(),
		}
		if err := msg.SetBlocks(dm.Blocks); err != nil {
			logger.Default().Error("set message blocks", zap.Error(err))
		}
		msgs = append(msgs, msg)
	}
	return conversation_svc.Conversation().SaveMessages(ctx, convID, msgs)
}

// GetCurrentConversationID 获取当前会话ID
func (a *App) GetCurrentConversationID() int64 {
	return a.currentConversationID
}

// DetectLocalCLIs 检测本地 AI CLI 工具
func (a *App) DetectLocalCLIs() []ai.CLIInfo {
	return ai.DetectLocalCLIs()
}

// RespondPermission 前端响应权限确认请求（CLI 工具用）
func (a *App) RespondPermission(behavior, message string) {
	resp := ai.PermissionResponse{Behavior: behavior, Message: message}
	// Codex 工具确认走 confirmCh
	if p, ok := a.aiProvider.(*ai.LocalCLIProvider); ok {
		if srv := p.GetCodexServer(); srv != nil {
			srv.RespondConfirm(resp)
			return
		}
	}
	select {
	case a.permissionChan <- resp:
	default:
	}
}

// ResetAISession 重置 AI 会话（创建新会话）
func (a *App) ResetAISession() {
	if p, ok := a.aiProvider.(*ai.LocalCLIProvider); ok {
		p.ResetSession()
	}
	a.currentConversationID = 0
}
