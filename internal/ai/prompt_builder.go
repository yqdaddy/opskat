package ai

import (
	"fmt"
	"strings"
)

// TabInfo 当前打开的 Tab 信息
type TabInfo struct {
	Type      string `json:"type"` // "ssh" | "database" | "redis" | "sftp"
	AssetID   int64  `json:"assetId"`
	AssetName string `json:"assetName"`
}

// AIContext 前端传入的上下文信息
type AIContext struct {
	OpenTabs []TabInfo `json:"openTabs"`
}

// PromptBuilder 动态构建 System Prompt
type PromptBuilder struct {
	language string
	context  AIContext
}

// NewPromptBuilder 创建 PromptBuilder
func NewPromptBuilder(language string, context AIContext) *PromptBuilder {
	return &PromptBuilder{
		language: language,
		context:  context,
	}
}

// Build 构建完整的 System Prompt
func (b *PromptBuilder) Build() string {
	var parts []string

	// 1. 基础角色描述
	parts = append(parts, b.buildRoleDescription())

	// 2. 用户语言
	parts = append(parts, b.buildLanguageHint())

	// 3. 当前 Tab 上下文
	if tabContext := b.buildTabContext(); tabContext != "" {
		parts = append(parts, tabContext)
	}

	// 4. 资产知识引导
	parts = append(parts, b.buildKnowledgeGuidance())

	// 5. 错误恢复引导
	parts = append(parts, b.buildErrorRecoveryGuidance())

	return strings.Join(parts, "\n\n")
}

func (b *PromptBuilder) buildRoleDescription() string {
	return `You are the OpsKat AI assistant, a powerful IT operations agent. You can:
- List, view, add, and update remote server assets (SSH, databases, Redis)
- Execute commands on SSH servers
- Execute SQL queries on databases (MySQL, PostgreSQL)
- Execute Redis commands
- Upload and download files via SFTP
- Request command execution permissions from the user
- Spawn sub-agents for complex multi-step tasks
- Execute batch commands across multiple assets simultaneously

You are proactive, thorough, and safety-conscious. Always verify before destructive operations.`
}

func (b *PromptBuilder) buildLanguageHint() string {
	switch b.language {
	case "zh-cn":
		return "请使用中文回复用户。"
	default:
		return "Respond in the same language the user uses."
	}
}

func (b *PromptBuilder) buildTabContext() string {
	if len(b.context.OpenTabs) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "The user currently has these tabs open (this helps you understand what they're working on):")
	for _, tab := range b.context.OpenTabs {
		typeName := tab.Type
		switch tab.Type {
		case "ssh":
			typeName = "SSH Terminal"
		case "database":
			typeName = "Database Query"
		case "redis":
			typeName = "Redis"
		case "sftp":
			typeName = "SFTP"
		}
		lines = append(lines, fmt.Sprintf("- %s: \"%s\" (ID: %d)", typeName, tab.AssetName, tab.AssetID))
	}
	return strings.Join(lines, "\n")
}

func (b *PromptBuilder) buildKnowledgeGuidance() string {
	return `When you discover valuable information about an asset during operations (OS version, running services, hardware specs, database version, etc.), proactively call update_asset to append these findings to the asset's Description field. When reading asset info, check the Description for existing knowledge to avoid redundant exploration.`
}

func (b *PromptBuilder) buildErrorRecoveryGuidance() string {
	return `When a tool execution fails, analyze the error, and try a different approach. If repeated attempts fail, explain the issue to the user and suggest alternatives. Do not give up after a single failure.`
}
