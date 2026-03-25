package ai

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"

	"github.com/opskat/opskat/internal/model/entity/asset_entity"
	"github.com/opskat/opskat/internal/service/asset_svc"
	"github.com/opskat/opskat/internal/service/credential_resolver"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func handleRequestGrant(ctx context.Context, args map[string]any) (string, error) {
	assetID := argInt64(args, "asset_id")
	commandPatterns := argString(args, "command_patterns")
	reason := argString(args, "reason")
	if assetID == 0 {
		return "", fmt.Errorf("缺少参数 asset_id")
	}
	if commandPatterns == "" {
		return "", fmt.Errorf("缺少参数 command_patterns")
	}

	// 按行拆分模式
	var patterns []string
	for _, line := range strings.Split(commandPatterns, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			patterns = append(patterns, line)
		}
	}
	if len(patterns) == 0 {
		return "", fmt.Errorf("command_patterns 不能为空")
	}

	checker := GetPolicyChecker(ctx)
	if checker == nil {
		return "", fmt.Errorf("权限检查器不可用")
	}

	result := checker.SubmitGrant(ctx, assetID, patterns, reason)
	setCheckResult(ctx, result)
	return result.Message, nil
}

func handleRunCommand(ctx context.Context, args map[string]any) (string, error) {
	assetID := argInt64(args, "asset_id")
	command := argString(args, "command")
	if assetID == 0 {
		return "", fmt.Errorf("缺少参数 asset_id")
	}
	if command == "" {
		return "", fmt.Errorf("缺少参数 command")
	}

	// 权限检查（两条路径共用）
	if checker := GetPolicyChecker(ctx); checker != nil {
		result := checker.Check(ctx, assetID, command)
		setCheckResult(ctx, result)
		if result.Decision != Allow {
			return result.Message, nil // 返回提示消息给 AI（非 error）
		}
	}

	asset, err := asset_svc.Asset().Get(ctx, assetID)
	if err != nil {
		return "", fmt.Errorf("资产不存在: %w", err)
	}
	if !asset.IsSSH() {
		return "", fmt.Errorf("资产不是SSH类型")
	}
	sshCfg, err := asset.GetSSHConfig()
	if err != nil {
		return "", fmt.Errorf("获取SSH配置失败: %w", err)
	}

	// 如果有 SSH 缓存（内置 Agent 模式），使用缓存连接
	if cache := getSSHCache(ctx); cache != nil {
		return runCommandWithCache(ctx, cache, assetID, sshCfg, command)
	}

	// 无缓存，创建一次性连接
	password, key, err := credential_resolver.Default().ResolveSSHCredentials(ctx, sshCfg)
	if err != nil {
		return "", fmt.Errorf("解析凭据失败: %w", err)
	}
	return executeSSHCommand(sshCfg, password, key, command)
}

func runCommandWithCache(ctx context.Context, cache *SSHClientCache, assetID int64, cfg *asset_entity.SSHConfig, command string) (string, error) {
	dial := func() (*ssh.Client, io.Closer, error) {
		password, key, err := credential_resolver.Default().ResolveSSHCredentials(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}
		client, err := createSSHClient(cfg, password, key)
		if err != nil {
			return nil, nil, err
		}
		return client, nil, nil
	}

	client, _, err := cache.GetOrDial(assetID, dial)
	if err != nil {
		return "", err
	}
	output, err := runSSHCommand(client, command)
	if err != nil {
		// 连接可能已断开，移除缓存后重试一次
		cache.Remove(assetID)
		client, _, err = cache.GetOrDial(assetID, dial)
		if err != nil {
			return "", err
		}
		output, err = runSSHCommand(client, command)
		if err != nil {
			cache.Remove(assetID)
			return "", err
		}
	}
	return output, nil
}

func handleUploadFile(ctx context.Context, args map[string]any) (string, error) {
	assetID := argInt64(args, "asset_id")
	localPath := argString(args, "local_path")
	remotePath := argString(args, "remote_path")
	if assetID == 0 || localPath == "" || remotePath == "" {
		return "", fmt.Errorf("缺少必要参数 (asset_id, local_path, remote_path)")
	}

	_, sshCfg, password, key, err := resolveAssetSSH(ctx, assetID)
	if err != nil {
		return "", err
	}

	err = executeWithSFTP(sshCfg, password, key, func(client *sftp.Client) error {
		srcFile, err := os.Open(localPath) //nolint:gosec
		if err != nil {
			return fmt.Errorf("打开本地文件失败: %w", err)
		}
		defer func() {
			if err := srcFile.Close(); err != nil {
				logger.Default().Warn("close local file", zap.String("path", localPath), zap.Error(err))
			}
		}()

		dstFile, err := client.Create(remotePath)
		if err != nil {
			return fmt.Errorf("创建远程文件失败: %w", err)
		}
		defer func() {
			if err := dstFile.Close(); err != nil {
				logger.Default().Warn("close remote file", zap.String("path", remotePath), zap.Error(err))
			}
		}()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"message":"文件上传成功","remote_path":"%s"}`, remotePath), nil
}

func handleDownloadFile(ctx context.Context, args map[string]any) (string, error) {
	assetID := argInt64(args, "asset_id")
	remotePath := argString(args, "remote_path")
	localPath := argString(args, "local_path")
	if assetID == 0 || remotePath == "" || localPath == "" {
		return "", fmt.Errorf("缺少必要参数 (asset_id, remote_path, local_path)")
	}

	_, sshCfg, password, key, err := resolveAssetSSH(ctx, assetID)
	if err != nil {
		return "", err
	}

	err = executeWithSFTP(sshCfg, password, key, func(client *sftp.Client) error {
		srcFile, err := client.Open(remotePath)
		if err != nil {
			return fmt.Errorf("打开远程文件失败: %w", err)
		}
		defer func() {
			if err := srcFile.Close(); err != nil {
				logger.Default().Warn("close remote file", zap.String("path", remotePath), zap.Error(err))
			}
		}()

		dstFile, err := os.Create(localPath) //nolint:gosec
		if err != nil {
			return fmt.Errorf("创建本地文件失败: %w", err)
		}
		defer func() {
			if err := dstFile.Close(); err != nil {
				logger.Default().Warn("close local file", zap.String("path", localPath), zap.Error(err))
			}
		}()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"message":"文件下载成功","local_path":"%s"}`, localPath), nil
}
