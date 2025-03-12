package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// 收集环境信息用于诊断
func debugEnvironment() {
	logger.Println("======================== 环境诊断开始 ========================")

	// 系统信息
	logger.Printf("操作系统: %s, 架构: %s", runtime.GOOS, runtime.GOARCH)
	logger.Printf("工作目录: %s", getCurrentDir())
	logger.Printf("执行用户: %s (UID=%s, GID=%s)", getUsername(), getUserID(), getGroupID())

	// 环境变量
	logger.Println("环境变量:")
	for _, env := range os.Environ() {
		logger.Printf("  %s", env)
	}

	// 特定环境变量
	logger.Printf("HOME=%s", os.Getenv("HOME"))
	logger.Printf("PATH=%s", os.Getenv("PATH"))
	logger.Printf("SHELL=%s", os.Getenv("SHELL"))

	// rclone信息
	debugRclone()

	logger.Println("======================== 环境诊断完成 ========================")
}

// 调试rclone配置和执行
func debugRclone() {
	logger.Println("rclone诊断:")

	// 检查rclone是否可执行
	if !checkCommandExists("rclone") {
		logger.Println("  rclone命令不存在或不在PATH中")
		return
	}

	// 检查rclone版本
	cmd := exec.Command("rclone", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Printf("  rclone version执行失败: %v", err)
	} else {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			logger.Printf("  %s", lines[0]) // 只输出第一行版本信息
		}
	}

	// 检查rclone配置文件
	homeDir := os.Getenv("HOME")
	configPaths := []string{
		filepath.Join(homeDir, ".config/rclone/rclone.conf"),
		filepath.Join(homeDir, ".rclone.conf"),
		"/etc/rclone/rclone.conf",
	}

	logger.Println("  检查rclone配置文件:")
	foundConfig := false
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			logger.Printf("  ✓ 找到配置文件: %s", path)
			foundConfig = true
			// 检查文件权限
			fileInfo, _ := os.Stat(path)
			logger.Printf("    权限: %s", fileInfo.Mode())

			// 检查文件内容是否包含WebDAV配置
			content, err := os.ReadFile(path)
			if err != nil {
				logger.Printf("    无法读取配置文件: %v", err)
			} else {
				if strings.Contains(string(content), "["+config.WebDAV.Remote+"]") {
					logger.Printf("    ✓ 找到WebDAV远程配置: %s", config.WebDAV.Remote)
				} else {
					logger.Printf("    ✗ 未找到WebDAV远程配置: %s", config.WebDAV.Remote)
				}
			}
		}
	}

	if !foundConfig {
		logger.Println("  ✗ 未找到任何rclone配置文件")
	}

	// 列出所有远程
	cmd = exec.Command("rclone", "listremotes")
	output, err = cmd.CombinedOutput()
	if err != nil {
		logger.Printf("  rclone listremotes执行失败: %v", err)
	} else {
		if len(output) > 0 {
			logger.Printf("  可用远程配置:\n%s", string(output))
		} else {
			logger.Println("  没有找到任何远程配置")
		}
	}
}

// 获取当前工作目录
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("获取失败: %v", err)
	}
	return dir
}

// 获取用户名
func getUsername() string {
	cmd := exec.Command("id", "-un")
	output, err := cmd.Output()
	if err != nil {
		return "未知"
	}
	return strings.TrimSpace(string(output))
}

// 获取用户ID
func getUserID() string {
	cmd := exec.Command("id", "-u")
	output, err := cmd.Output()
	if err != nil {
		return "未知"
	}
	return strings.TrimSpace(string(output))
}

// 获取组ID
func getGroupID() string {
	cmd := exec.Command("id", "-g")
	output, err := cmd.Output()
	if err != nil {
		return "未知"
	}
	return strings.TrimSpace(string(output))
}
