package ai

// AgentConfig Agent 配置
type AgentConfig struct {
	MaxRounds    int       // 最大对话轮次，0 表示使用默认值
	MaxResultLen int       // 工具结果最大长度，0 表示使用默认值
	Tools        []ToolDef // 可用工具列表，nil 表示全部
	SystemPrompt string    // System Prompt，空表示不注入
	IsSubAgent   bool      // 是否为 Sub Agent（限制嵌套）
}

const (
	DefaultMainAgentRounds = 50
	DefaultSubAgentRounds  = 30
	MaxAbsoluteRounds      = 100
	DefaultMaxResultLen    = 32 * 1024
)

// NewDefaultConfig 创建主 Agent 默认配置
func NewDefaultConfig() AgentConfig {
	return AgentConfig{
		MaxRounds:    DefaultMainAgentRounds,
		MaxResultLen: DefaultMaxResultLen,
	}
}

// NewSubAgentConfig 创建 Sub Agent 默认配置
func NewSubAgentConfig() AgentConfig {
	return AgentConfig{
		MaxRounds:    DefaultSubAgentRounds,
		MaxResultLen: DefaultMaxResultLen,
		IsSubAgent:   true,
	}
}

func (c AgentConfig) effectiveMaxRounds() int {
	if c.MaxRounds <= 0 {
		return DefaultMainAgentRounds
	}
	if c.MaxRounds > MaxAbsoluteRounds {
		return MaxAbsoluteRounds
	}
	return c.MaxRounds
}

func (c AgentConfig) effectiveMaxResultLen() int {
	if c.MaxResultLen <= 0 {
		return DefaultMaxResultLen
	}
	return c.MaxResultLen
}
