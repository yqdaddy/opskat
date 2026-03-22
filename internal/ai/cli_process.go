package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// CLIProcess 管理 CLI 子进程的生命周期和 stdin/stdout 通信
type CLIProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	mu     sync.Mutex
}

// StartCLIProcess 启动 CLI 子进程
func StartCLIProcess(ctx context.Context, cliPath string, args []string) (*CLIProcess, error) {
	cmd := exec.CommandContext(ctx, cliPath, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("获取 stdin 失败: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("获取 stdout 失败: %w", err)
	}
	// stderr 丢弃，避免阻塞
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("启动 CLI 失败: %w", err)
	}

	return &CLIProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

// WriteJSON 向 stdin 写入一行 JSON（NDJSON 格式）
func (p *CLIProcess) WriteJSON(v any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %w", err)
	}
	data = append(data, '\n')
	_, err = p.stdin.Write(data)
	return err
}

// ReadLines 从 stdout 逐行读取，返回 channel。进程结束或 ctx 取消时关闭。
func (p *CLIProcess) ReadLines(ctx context.Context) <-chan string {
	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(p.stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case ch <- scanner.Text():
			}
		}
	}()
	return ch
}

// Wait 等待进程结束
func (p *CLIProcess) Wait() error {
	return p.cmd.Wait()
}

// Stop 停止进程
func (p *CLIProcess) Stop() {
	p.stdin.Close()
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
}
