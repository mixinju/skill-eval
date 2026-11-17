package runner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type commandSpec struct {
	Bin      string
	Args     []string
	Dir      string
	Stdin    string
	ExtraEnv []string
}

type commandFunc func(ctx context.Context, spec commandSpec) (stdout, stderr string, exitCode int, err error)

func realCommand(ctx context.Context, spec commandSpec) (string, string, int, error) {
	if spec.Bin == "" {
		return "", "", -1, fmt.Errorf("未指定可执行文件")
	}
	cmd := exec.CommandContext(ctx, spec.Bin, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = append(os.Environ(), spec.ExtraEnv...)
	if spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", -1, fmt.Errorf("打开 %s 标准输出失败: %w", spec.Bin, err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", stderr.String(), -1, fmt.Errorf("启动 %s 失败: %w。请确认 CLI 已安装且当前环境有权限执行", spec.Bin, err)
	}

	var stdout bytes.Buffer
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		stdout.WriteString(scanner.Text())
		stdout.WriteByte('\n')
	}
	scanErr := scanner.Err()
	// 把管道读完再 Wait，避免子进程因管道写满而卡住。
	_, _ = io.Copy(io.Discard, stdoutPipe)
	waitErr := cmd.Wait()

	exit := -1
	if cmd.ProcessState != nil {
		exit = cmd.ProcessState.ExitCode()
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return stdout.String(), stderr.String(), exit, fmt.Errorf("执行超时")
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return stdout.String(), stderr.String(), exit, fmt.Errorf("任务已取消")
	}
	if scanErr != nil {
		return stdout.String(), stderr.String(), exit, fmt.Errorf("读取 %s 输出失败: %w", spec.Bin, scanErr)
	}
	if waitErr != nil {
		return stdout.String(), stderr.String(), exit, fmt.Errorf("%s 退出状态 %d: %w", filepathBase(spec.Bin), exit, waitErr)
	}
	return stdout.String(), stderr.String(), exit, nil
}

func filepathBase(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}

func lookupBin(envKey, name string) (string, error) {
	if p := strings.TrimSpace(os.Getenv(envKey)); p != "" {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("未找到 %s 可执行文件：环境变量 %s=%s 无效。请安装 CLI 或修正路径，参见 docs/runners.md", name, envKey, p)
		}
		return p, nil
	}
	p, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("未找到 %s 可执行文件。请安装 CLI 并加入 PATH，或设置 %s。参见 docs/runners.md", name, envKey)
	}
	return p, nil
}

func commandVersion(ctx context.Context, bin string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(out))
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	if len(line) > 160 {
		line = line[:160]
	}
	return line
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func quoteArgs(args []string) string {
	return strings.Join(args, " ")
}
