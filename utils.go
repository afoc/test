package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// ================ 命令执行辅助 ================

// runCmdSilent 执行命令并静默丢弃 stdout/stderr（仅返回错误）
func runCmdSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// runCmdCombined 执行命令并捕获 stdout+stderr（避免泄露到终端）
func runCmdCombined(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// containsString 检查字符串是否包含子串
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// ================ PID 文件管理 ================

// createPIDFile 创建PID文件（防止多实例运行）
func createPIDFile(pidFile string) error {
	// 检查PID文件是否已存在
	if data, err := os.ReadFile(pidFile); err == nil {
		// PID文件存在，检查进程是否还在运行
		var oldPID int
		if _, err := fmt.Sscanf(string(data), "%d", &oldPID); err == nil {
			// 检查进程是否存在
			process, err := os.FindProcess(oldPID)
			if err == nil {
				// 尝试发送信号0来检查进程是否存在
				if err := process.Signal(syscall.Signal(0)); err == nil {
					return fmt.Errorf("程序已在运行 (PID: %d)，如果确认没有运行，请删除: %s", oldPID, pidFile)
				}
			}
			// 进程不存在，清理旧的PID文件
			log.Printf("检测到旧的PID文件 (PID: %d 已不存在)，正在清理...", oldPID)
			os.Remove(pidFile)
		}
	}

	// 写入当前进程PID
	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
		return fmt.Errorf("创建PID文件失败: %v", err)
	}

	log.Printf("已创建PID文件: %s (PID: %d)", pidFile, pid)
	return nil
}

// removePIDFile 删除PID文件
func removePIDFile(pidFile string) {
	if err := os.Remove(pidFile); err != nil {
		log.Printf("警告: 删除PID文件失败: %v", err)
	} else {
		log.Printf("已删除PID文件: %s", pidFile)
	}
}

// ================ 格式化工具 ================

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatDuration 格式化时间
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d分%d秒", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%d时%d分", int(d.Hours()), int(d.Minutes())%60)
	}
}
