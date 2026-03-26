package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type batchCommandItem struct {
	Asset   string `json:"asset"`
	Type    string `json:"type"`
	Command string `json:"command"`
}

type batchResultItem struct {
	AssetID   int64  `json:"asset_id"`
	AssetName string `json:"asset_name"`
	Type      string `json:"type"`
	Command   string `json:"command"`
	ExitCode  int    `json:"exit_code"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr,omitempty"`
	Error     string `json:"error,omitempty"`
}

func handleBatchCommand(ctx context.Context, args map[string]any) (string, error) {
	commandsRaw, ok := args["commands"]
	if !ok {
		return "", fmt.Errorf("missing required parameter: commands")
	}

	commandsJSON, err := json.Marshal(commandsRaw)
	if err != nil {
		return "", fmt.Errorf("invalid commands parameter: %w", err)
	}
	var commands []batchCommandItem
	if err := json.Unmarshal(commandsJSON, &commands); err != nil {
		return "", fmt.Errorf("invalid commands format: %w", err)
	}

	if len(commands) == 0 {
		return "No commands to execute.", nil
	}

	for i := range commands {
		if commands[i].Type == "" {
			commands[i].Type = "exec"
		}
	}

	checker := GetPolicyChecker(ctx)

	// 解析资产并预检策略
	type resolvedCmd struct {
		item      batchCommandItem
		assetID   int64
		assetName string
		decision  string // "allow", "deny"
		denyMsg   string
	}
	var resolved []resolvedCmd

	for _, cmd := range commands {
		assetID, assetName, resolveErr := resolveAssetForBatch(ctx, cmd.Asset)
		if resolveErr != nil {
			resolved = append(resolved, resolvedCmd{
				item: cmd, decision: "deny", denyMsg: fmt.Sprintf("asset not found: %s", cmd.Asset),
			})
			continue
		}

		decision := "allow"
		denyMsg := ""

		// 使用 CheckForAsset 处理完整的权限检查流程（包含用户确认）
		if checker != nil {
			result := checker.CheckForAsset(ctx, assetID, cmd.Type, cmd.Command)
			switch result.Decision {
			case Deny:
				decision = "deny"
				denyMsg = result.Message
			case Allow:
				decision = "allow"
			default:
				decision = "allow"
			}
		}

		resolved = append(resolved, resolvedCmd{
			item: cmd, assetID: assetID, assetName: assetName,
			decision: decision, denyMsg: denyMsg,
		})
	}

	// 分桶
	var approved, denied []resolvedCmd
	for _, r := range resolved {
		switch r.decision {
		case "allow":
			approved = append(approved, r)
		case "deny":
			denied = append(denied, r)
		}
	}

	// 并行执行（max 10 并发）
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	var mu sync.Mutex
	var results []batchResultItem

	for _, r := range denied {
		results = append(results, batchResultItem{
			AssetID: r.assetID, AssetName: r.assetName,
			Type: r.item.Type, Command: r.item.Command,
			ExitCode: -1, Error: fmt.Sprintf("denied: %s", r.denyMsg),
		})
	}

	var wg sync.WaitGroup
	for _, r := range approved {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := executeBatchItem(ctx, r.item, r.assetID, r.assetName)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}
	wg.Wait()

	output, err := json.MarshalIndent(map[string]any{"results": results}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func executeBatchItem(ctx context.Context, item batchCommandItem, assetID int64, assetName string) batchResultItem {
	result := batchResultItem{
		AssetID: assetID, AssetName: assetName,
		Type: item.Type, Command: item.Command,
	}

	var handlerName string
	var handlerArgs map[string]any

	switch item.Type {
	case "exec":
		handlerName = "run_command"
		handlerArgs = map[string]any{"asset_id": assetID, "command": item.Command}
	case "sql":
		handlerName = "exec_sql"
		handlerArgs = map[string]any{"asset_id": assetID, "sql": item.Command}
	case "redis":
		handlerName = "exec_redis"
		handlerArgs = map[string]any{"asset_id": assetID, "command": item.Command}
	default:
		result.ExitCode = -1
		result.Error = fmt.Sprintf("unknown type: %s", item.Type)
		return result
	}

	handler := getToolHandler(handlerName)
	if handler == nil {
		result.ExitCode = -1
		result.Error = fmt.Sprintf("handler not found: %s", handlerName)
		return result
	}

	output, err := handler(ctx, handlerArgs)
	if err != nil {
		result.ExitCode = -1
		result.Error = err.Error()
	} else {
		result.ExitCode = 0
		result.Stdout = output
	}
	return result
}

func resolveAssetForBatch(ctx context.Context, assetRef string) (int64, string, error) {
	handler := getToolHandler("get_asset")
	if handler == nil {
		return 0, "", fmt.Errorf("get_asset handler not found")
	}

	result, err := handler(ctx, map[string]any{"id": assetRef})
	if err != nil {
		return 0, "", err
	}

	var asset struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(result), &asset); err != nil {
		return 0, "", fmt.Errorf("cannot resolve asset: %s", assetRef)
	}
	return asset.ID, asset.Name, nil
}

func getToolHandler(name string) ToolHandlerFunc {
	for _, def := range AllToolDefs() {
		if def.Name == name {
			return def.Handler
		}
	}
	return nil
}
