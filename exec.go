package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

func (mw *MyMainWindow) ShellExecWithArgs(ctx context.Context, cmdName string, args []string, dir string, timeout time.Duration) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW | windows.CREATE_NEW_PROCESS_GROUP,
	}

	slog.Info(fmt.Sprintf("cmd: %s %v, path: %s", cmdName, args, cmd.Dir))

	return cmd, mw.executeCommand(ctx, cmd, timeout)
}

func (mw *MyMainWindow) executeCommand(ctx context.Context, cmd *exec.Cmd, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	//timer := time.AfterFunc(timeout, cancel)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("StderrPipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cmd.Start: %w", err)
	}
	// ログをリアルタイム出力
	go mw.streamLogs(stdout, false)
	go mw.streamLogs(stderr, true)
	if isStart {
		mainProcess = cmd
	}

	err = cmd.Wait()
	return err
}
// 標準出力・エラーをリアルタイムで TextEdit & slog に反映
func (mw *MyMainWindow) streamLogs(reader io.Reader, isError bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logMsg := strings.TrimSpace(scanner.Text())
		mw.logToUI(logMsg)

		// エラーメッセージなら slog.Error を使用
		if isError {
			slog.Error(logMsg)
		} else {
			slog.Info(logMsg)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Log stream error: " + err.Error())
	}
}

// walk.TextEdit にログを追加（UI スレッドで実行）
func (mw *MyMainWindow) logToUI(text string) {
	mw.Synchronize(func() {
		mw.logTE.AppendText(text + "\r\n")
	})
}

// プロセスグループごと強制終了する関数
func (mw *MyMainWindow) killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// `TASKKILL` コマンドを使い、プロセスグループごと終了させる
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
	killCmd.SysProcAttr = &windows.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	return killCmd.Run()
}
