package app

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/opskat/opskat/internal/ai"
	"github.com/opskat/opskat/internal/approval"
	"github.com/opskat/opskat/internal/bootstrap"
	"github.com/opskat/opskat/internal/model/entity/grant_entity"
	"github.com/opskat/opskat/internal/repository/grant_repo"
	"github.com/opskat/opskat/internal/sshpool"

	"github.com/cago-frame/cago/pkg/logger"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

// startApprovalServer 启动 opsctl 审批 Unix socket 服务
func (a *App) startApprovalServer(authToken string) {
	handler := func(req approval.ApprovalRequest) approval.ApprovalResponse {
		// 数据变更通知：opsctl 通知前端刷新
		if req.Type == "notify" {
			wailsRuntime.EventsEmit(a.ctx, "data:changed", map[string]any{
				"resource": req.Detail,
			})
			return approval.ApprovalResponse{Approved: true}
		}

		// 授权审批
		if req.Type == "grant" {
			return a.handleGrantApproval(req)
		}

		// 单条审批
		confirmID := fmt.Sprintf("opsctl_%d", time.Now().UnixNano())

		// 激活应用窗口
		a.activateWindow()

		wailsRuntime.EventsEmit(a.ctx, "opsctl:approval", map[string]any{
			"confirm_id": confirmID,
			"type":       req.Type,
			"asset_id":   req.AssetID,
			"asset_name": req.AssetName,
			"command":    req.Command,
			"detail":     req.Detail,
			"session_id": req.SessionID,
		})

		ch := make(chan bool, 1)
		a.pendingApprovals.Store(confirmID, ch)
		defer a.pendingApprovals.Delete(confirmID)

		select {
		case approved := <-ch:
			if approved {
				return approval.ApprovalResponse{Approved: true}
			}
			return approval.ApprovalResponse{Approved: false, Reason: "user denied"}
		case <-a.ctx.Done():
			return approval.ApprovalResponse{Approved: false, Reason: "app shutting down"}
		case <-a.shutdownCh:
			return approval.ApprovalResponse{Approved: false, Reason: "app shutting down"}
		}
	}

	srv := approval.NewServer(handler, authToken)
	sockPath := approval.SocketPath(bootstrap.AppDataDir())
	if err := srv.Start(sockPath); err != nil {
		log.Printf("Approval server failed to start: %v", err)
		return
	}
	a.approvalServer = srv
}

// startSSHPoolServer 启动 SSH 连接池 proxy 服务
func (a *App) startSSHPoolServer(authToken string) {
	dialer := &appPoolDialer{sshManager: a.sshManager}
	a.sshPool = sshpool.NewPool(dialer, 5*time.Minute)
	a.sshProxyServer = sshpool.NewServer(a.sshPool, authToken)
	sockPath := sshpool.SocketPath(bootstrap.AppDataDir())
	if err := a.sshProxyServer.Start(sockPath); err != nil {
		log.Printf("SSH pool server failed to start: %v", err)
		return
	}
}

// handleGrantApproval 处理批量计划审批
func (a *App) handleGrantApproval(req approval.ApprovalRequest) approval.ApprovalResponse {
	ctx := a.langCtx()
	sessionID := req.SessionID

	// 写入 DB
	session := &grant_entity.GrantSession{
		ID:          sessionID,
		Description: req.Description,
		Status:      grant_entity.GrantStatusPending,
		Createtime:  time.Now().Unix(),
	}
	if err := grant_repo.Grant().CreateSession(ctx, session); err != nil {
		// Session may already exist (e.g., multiple request_permission calls in same conversation)
		if _, getErr := grant_repo.Grant().GetSession(ctx, sessionID); getErr != nil {
			return approval.ApprovalResponse{Approved: false, Reason: "failed to create grant session"}
		}
	}

	var items []*grant_entity.GrantItem
	for i, pi := range req.GrantItems {
		items = append(items, &grant_entity.GrantItem{
			GrantSessionID: sessionID,
			ItemIndex:      i,
			ToolName:       pi.Type,
			AssetID:        pi.AssetID,
			AssetName:      pi.AssetName,
			GroupID:        pi.GroupID,
			GroupName:      pi.GroupName,
			Command:        pi.Command,
			Detail:         pi.Detail,
		})
	}
	if err := grant_repo.Grant().CreateItems(ctx, items); err != nil {
		return approval.ApprovalResponse{Approved: false, Reason: "failed to create grant items"}
	}

	// 构建前端事件数据
	eventItems := make([]map[string]any, 0, len(req.GrantItems))
	for _, pi := range req.GrantItems {
		eventItems = append(eventItems, map[string]any{
			"type":       pi.Type,
			"asset_id":   pi.AssetID,
			"asset_name": pi.AssetName,
			"group_id":   pi.GroupID,
			"group_name": pi.GroupName,
			"command":    pi.Command,
			"detail":     pi.Detail,
		})
	}

	// 激活应用窗口
	a.activateWindow()

	wailsRuntime.EventsEmit(a.ctx, "opsctl:grant-approval", map[string]any{
		"session_id":  sessionID,
		"description": req.Description,
		"items":       eventItems,
	})

	// 等待前端响应
	ch := make(chan bool, 1)
	a.pendingApprovals.Store(sessionID, ch)
	defer a.pendingApprovals.Delete(sessionID)

	select {
	case approved := <-ch:
		if approved {
			if err := grant_repo.Grant().UpdateSessionStatus(ctx, sessionID, grant_entity.GrantStatusApproved); err != nil {
				logger.Default().Error("update grant session status to approved", zap.Error(err))
			}
			resp := approval.ApprovalResponse{Approved: true, SessionID: sessionID}
			// 读取最终的 items（可能已被用户编辑）
			if finalItems, err := grant_repo.Grant().ListItems(ctx, sessionID); err == nil {
				for _, item := range finalItems {
					resp.EditedItems = append(resp.EditedItems, approval.GrantItem{
						Type:      item.ToolName,
						AssetID:   item.AssetID,
						AssetName: item.AssetName,
						GroupID:   item.GroupID,
						GroupName: item.GroupName,
						Command:   item.Command,
						Detail:    item.Detail,
					})
				}
			}
			return resp
		}
		if err := grant_repo.Grant().UpdateSessionStatus(ctx, sessionID, grant_entity.GrantStatusRejected); err != nil {
			logger.Default().Error("update grant session status to rejected", zap.Error(err))
		}
		return approval.ApprovalResponse{Approved: false, Reason: "user denied", SessionID: sessionID}
	case <-a.ctx.Done():
		if err := grant_repo.Grant().UpdateSessionStatus(ctx, sessionID, grant_entity.GrantStatusRejected); err != nil {
			logger.Default().Error("update grant session status to rejected on shutdown", zap.Error(err))
		}
		return approval.ApprovalResponse{Approved: false, Reason: "app shutting down"}
	case <-a.shutdownCh:
		if err := grant_repo.Grant().UpdateSessionStatus(ctx, sessionID, grant_entity.GrantStatusRejected); err != nil {
			logger.Default().Error("update grant session status to rejected on shutdown", zap.Error(err))
		}
		return approval.ApprovalResponse{Approved: false, Reason: "app shutting down"}
	}
}

// RespondOpsctlApproval 前端响应 opsctl 审批请求
func (a *App) RespondOpsctlApproval(confirmID string, approved bool) {
	if v, ok := a.pendingApprovals.Load(confirmID); ok {
		ch := v.(chan bool)
		select {
		case ch <- approved:
		default:
		}
	}
}

// GrantItemEdit 前端编辑后的 grant item
type GrantItemEdit struct {
	AssetID   int64  `json:"asset_id"`
	AssetName string `json:"asset_name"`
	GroupID   int64  `json:"group_id"`
	GroupName string `json:"group_name"`
	Command   string `json:"command"`
}

// RespondGrantApproval 前端响应计划审批请求
func (a *App) RespondGrantApproval(sessionID string, approved bool) {
	a.RespondOpsctlApproval(sessionID, approved)
}

// RespondGrantApprovalWithEdits 前端响应计划审批请求并更新编辑后的 items
func (a *App) RespondGrantApprovalWithEdits(sessionID string, approved bool, editedItems []GrantItemEdit) {
	if approved && len(editedItems) > 0 {
		// 更新 grant items
		var items []*grant_entity.GrantItem
		for i, edit := range editedItems {
			// 支持一行多个命令（换行分隔）
			lines := strings.Split(edit.Command, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				items = append(items, &grant_entity.GrantItem{
					GrantSessionID: sessionID,
					ItemIndex:      i,
					ToolName:       "exec",
					AssetID:        edit.AssetID,
					AssetName:      edit.AssetName,
					GroupID:        edit.GroupID,
					GroupName:      edit.GroupName,
					Command:        line,
				})
			}
		}
		if len(items) > 0 {
			if err := grant_repo.Grant().UpdateItems(a.langCtx(), sessionID, items); err != nil {
				logger.Default().Error("update grant items", zap.Error(err))
			}
		}
	}
	a.RespondOpsctlApproval(sessionID, approved)
}

// RespondOpsctlApprovalGrant 前端响应审批并记住 grant 命令模式
func (a *App) RespondOpsctlApprovalGrant(confirmID string, approved bool, sessionID string, assetID int64, assetName string, commandPattern string) {
	if approved && sessionID != "" && commandPattern != "" {
		ai.SaveGrantPattern(a.langCtx(), sessionID, assetID, assetName, commandPattern)
	}
	a.RespondOpsctlApproval(confirmID, approved)
}

// makeCommandConfirmFunc 创建命令确认回调，向 AI 聊天流发送 tool_confirm 事件并阻塞等待
func (a *App) makeCommandConfirmFunc() ai.CommandConfirmFunc {
	return func(assetName, command string) (bool, bool) {
		convID := a.currentConversationID
		confirmID := fmt.Sprintf("cmd_%d_%d", convID, time.Now().UnixNano())
		eventName := fmt.Sprintf("ai:event:%d", convID)

		// 向 AI 聊天流发送 tool_confirm 事件
		wailsRuntime.EventsEmit(a.ctx, eventName, ai.StreamEvent{
			Type:      "tool_confirm",
			ToolName:  "run_command",
			ToolInput: fmt.Sprintf("[%s] $ %s", assetName, command),
			ConfirmID: confirmID,
		})

		// 阻塞等待前端响应
		ch := make(chan ConfirmResponse, 1)
		a.pendingConfirms.Store(confirmID, ch)
		defer a.pendingConfirms.Delete(confirmID)

		select {
		case resp := <-ch:
			// 发送确认结果事件更新 UI 状态
			wailsRuntime.EventsEmit(a.ctx, eventName, ai.StreamEvent{
				Type:      "tool_confirm_result",
				ConfirmID: confirmID,
				Content:   resp.Behavior,
			})
			return resp.Behavior != "deny", resp.Behavior == "allowAll"
		case <-a.ctx.Done():
			return false, false
		case <-a.shutdownCh:
			return false, false
		}
	}
}

// makeGrantRequestFunc 创建 Grant 审批回调，复用 grant 审批弹窗
func (a *App) makeGrantRequestFunc() ai.GrantRequestFunc {
	return func(assetID int64, assetName string, patterns []string, reason string) (bool, []string) {
		// 构建 ApprovalRequest 并走 grant 审批流程
		grantItems := make([]approval.GrantItem, 0, len(patterns))
		for _, p := range patterns {
			grantItems = append(grantItems, approval.GrantItem{
				Type:      "exec",
				AssetID:   assetID,
				AssetName: assetName,
				Command:   p,
				Detail:    reason,
			})
		}

		resp := a.handleGrantApproval(approval.ApprovalRequest{
			Type:        "grant",
			SessionID:   fmt.Sprintf("conv_%d", a.currentConversationID),
			GrantItems:  grantItems,
			Description: reason,
		})

		if !resp.Approved {
			return false, nil
		}

		// 读回可能被用户编辑过的 items
		items, err := grant_repo.Grant().ListItems(a.langCtx(), resp.SessionID)
		if err != nil || len(items) == 0 {
			return true, patterns
		}
		var finalPatterns []string
		for _, item := range items {
			finalPatterns = append(finalPatterns, item.Command)
		}
		return true, finalPatterns
	}
}

// RespondCommandConfirm 前端响应 run_command 确认请求
func (a *App) RespondCommandConfirm(confirmID, behavior string) {
	// 先检查普通命令确认（有明确的 confirmID 匹配）
	if v, ok := a.pendingConfirms.Load(confirmID); ok {
		ch := v.(chan ConfirmResponse)
		select {
		case ch <- ConfirmResponse{Behavior: behavior}:
		default:
		}
		return
	}
	// 否则转发到 Codex 工具确认
	if p, ok := a.aiProvider.(*ai.LocalCLIProvider); ok {
		if srv := p.GetCodexServer(); srv != nil {
			srv.RespondConfirm(ai.PermissionResponse{Behavior: behavior})
		}
	}
}
