package ai

import (
	"fmt"
	"strings"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"

	"mvdan.cc/sh/v3/syntax"
)

// --- Shell AST 解析 ---

// ExtractSubCommands 从 shell 命令中提取所有子命令（处理 &&、||、;、|、$() 等）
func ExtractSubCommands(command string) ([]string, error) {
	parser := syntax.NewParser()
	file, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, fmt.Errorf("shell 解析失败: %w", err)
	}

	var cmds []string
	printer := syntax.NewPrinter()

	var extractFromStmt func(stmt *syntax.Stmt)
	extractFromStmt = func(stmt *syntax.Stmt) {
		if stmt == nil || stmt.Cmd == nil {
			return
		}
		switch cmd := stmt.Cmd.(type) {
		case *syntax.BinaryCmd:
			// &&、||、| 等二元操作
			extractFromStmt(cmd.X)
			extractFromStmt(cmd.Y)
		default:
			// CallExpr、其他命令类型 — 打印为字符串
			var buf strings.Builder
			if err := printer.Print(&buf, stmt.Cmd); err != nil {
				logger.Default().Warn("print shell statement", zap.Error(err))
			}
			cmdStr := strings.TrimSpace(buf.String())
			if cmdStr != "" {
				cmds = append(cmds, cmdStr)
			}
		}
	}

	syntax.Walk(file, func(node syntax.Node) bool {
		stmt, ok := node.(*syntax.Stmt)
		if !ok {
			return true
		}
		extractFromStmt(stmt)
		return false
	})

	return cmds, nil
}
