package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WebDAV struct {
		Remote        string `yaml:"remote"`
		Path          string `yaml:"path"`
		RetentionDays int    `yaml:"retention_days"`
		RcloneConfig  string `yaml:"rclone_config"`
	} `yaml:"webdav"`

	MySQL struct {
		Enabled   bool     `yaml:"enabled"`
		Host      string   `yaml:"host"`
		Port      int      `yaml:"port"`
		User      string   `yaml:"user"`
		Password  string   `yaml:"password"`
		Databases []string `yaml:"databases"`
	} `yaml:"mysql"`

	Docker struct {
		Enabled       bool     `yaml:"enabled"`
		Containers    []string `yaml:"containers"`
		BackupCompose bool     `yaml:"backup_compose"`
		ComposePaths  []string `yaml:"compose_paths"`
		BackupVolumes bool     `yaml:"backup_volumes"`
	} `yaml:"docker"`

	Files struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"paths"`
	} `yaml:"files"`

	Logging struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logging"`
}

var (
	config  Config
	logger  *log.Logger
	tempDir = "/tmp/backup"
	dateStr = time.Now().Format("20060102_150405")
	help    bool
)

// 检查命令是否存在
func checkCommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// 检查备份所需的依赖
func checkDependencies() error {
	var missingDeps []string

	// 检查MySQL备份依赖
	if config.MySQL.Enabled {
		if !checkCommandExists("mysqldump") {
			missingDeps = append(missingDeps, "mysqldump (用于MySQL备份)")
		}
		if !checkCommandExists("gzip") {
			missingDeps = append(missingDeps, "gzip (用于压缩MySQL备份)")
		}
		if !checkCommandExists("mysql") {
			missingDeps = append(missingDeps, "mysql (用于测试数据库连接)")
		} else {
			// 测试MySQL连接
			testArgs := []string{
				"-h" + config.MySQL.Host,
				fmt.Sprintf("-P%d", config.MySQL.Port),
				"-u" + config.MySQL.User,
			}
			// 只有在密码非空时才添加密码参数
			if config.MySQL.Password != "" {
				testArgs = append(testArgs, "-p"+config.MySQL.Password)
			}
			// 如果指定了端口，强制使用TCP协议
			if config.MySQL.Port > 0 {
				testArgs = append(testArgs, "--protocol=tcp")
			}
			testArgs = append(testArgs, "-e", "SELECT 1")

			if err := exec.Command("mysql", testArgs...).Run(); err != nil {
				//调试输出命令行完整指令
				cmdStr := fmt.Sprintf("mysql %v\n", strings.Join(testArgs, " "))
				missingDeps = append(missingDeps, fmt.Sprintf("MySQL连接测试失败: %v. 完整命令: %s", err, cmdStr))
			}
		}
	}

	// 检查Docker备份依赖
	if config.Docker.Enabled {
		if !checkCommandExists("docker") {
			missingDeps = append(missingDeps, "docker")
		} else {
			// 测试Docker守护进程
			if err := exec.Command("docker", "ps").Run(); err != nil {
				missingDeps = append(missingDeps, fmt.Sprintf("Docker服务未运行或权限不足: %v", err))
			}

			// 如果启用了compose备份，检查目录是否存在
			if config.Docker.BackupCompose {
				for _, pattern := range config.Docker.ComposePaths {
					matches, err := filepath.Glob(pattern)
					if err != nil {
						missingDeps = append(missingDeps, fmt.Sprintf("无效的compose路径模式 %s: %v", pattern, err))
					} else if len(matches) == 0 {
						missingDeps = append(missingDeps, fmt.Sprintf("未找到匹配的compose文件: %s", pattern))
					}
				}
			}
		}
	}

	// 检查文件备份依赖
	if config.Files.Enabled {
		if !checkCommandExists("tar") {
			missingDeps = append(missingDeps, "tar (用于文件备份)")
		}
		// 检查备份路径是否存在且可访问
		for _, path := range config.Files.Paths {
			parts := strings.Split(path, ":")
			if len(parts) != 2 {
				missingDeps = append(missingDeps, fmt.Sprintf("无效的文件备份路径格式: %s", path))
				continue
			}
			srcPath := parts[0]
			if _, err := os.Stat(srcPath); err != nil {
				missingDeps = append(missingDeps, fmt.Sprintf("无法访问备份路径 %s: %v", srcPath, err))
			}
		}
	}

	// 检查WebDAV上传依赖
	if !checkCommandExists("rclone") {
		missingDeps = append(missingDeps, "rclone (用于WebDAV上传)")
	} else {
		// 测试WebDAV配置
		testCmd := exec.Command("rclone", "lsd", fmt.Sprintf("%s:", config.WebDAV.Remote))
		if err := testCmd.Run(); err != nil {
			missingDeps = append(missingDeps, fmt.Sprintf("WebDAV配置无效或连接失败: %v", err))
		}
	}

	if len(missingDeps) > 0 {
		return fmt.Errorf("环境检查失败:\n%s", strings.Join(missingDeps, "\n"))
	}

	return nil
}

// 暂停需要备份的容器
func pauseContainers() []string {
	logger.Println("暂停容器以确保数据一致性...")

	var containersToResume []string
	containers := getDockerContainers()

	for _, container := range containers {
		logger.Printf("暂停容器: %s", container)
		if err := runCommand("docker", []string{"pause", container}, ""); err != nil {
			logger.Printf("无法暂停容器 %s: %v", container, err)
			continue
		}
		containersToResume = append(containersToResume, container)
	}

	logger.Printf("成功暂停 %d 个容器", len(containersToResume))
	return containersToResume
}

// 恢复暂停的容器
func resumeContainers(containers []string) {
	logger.Println("恢复暂停的容器...")

	for _, container := range containers {
		logger.Printf("恢复容器: %s", container)
		if err := runCommand("docker", []string{"unpause", container}, ""); err != nil {
			logger.Printf("无法恢复容器 %s: %v", container, err)
		}
	}

	logger.Println("所有容器已恢复运行")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\n示例:")
		fmt.Println("  omnibak -c /path/to/config.yaml")
	}

	// 解析命令行参数
	var configPath string
	flag.BoolVar(&help, "h", false, "显示帮助信息")
	flag.StringVar(&configPath, "c", "config.yaml", "配置文件路径")
	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("无法读取配置文件: %v", err)
	}

	// 设置默认值
	config = Config{
		MySQL: struct {
			Enabled   bool     `yaml:"enabled"`
			Host      string   `yaml:"host"`
			Port      int      `yaml:"port"`
			User      string   `yaml:"user"`
			Password  string   `yaml:"password"`
			Databases []string `yaml:"databases"`
		}{
			Enabled: false,
			Host:    "localhost",
			Port:    3306,
		},
		Docker: struct {
			Enabled       bool     `yaml:"enabled"`
			Containers    []string `yaml:"containers"`
			BackupCompose bool     `yaml:"backup_compose"`
			ComposePaths  []string `yaml:"compose_paths"`
			BackupVolumes bool     `yaml:"backup_volumes"`
		}{
			Enabled: false,
		},
		Files: struct {
			Enabled bool     `yaml:"enabled"`
			Paths   []string `yaml:"paths"`
		}{
			Enabled: false,
		},
		Logging: struct {
			Level string `yaml:"level"`
			File  string `yaml:"file"`
		}{
			Level: "info",
			File:  "omnibak.log",
		},
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("无法解析配置文件: %v", err)
	}

	// 初始化日志
	var logWriter io.Writer = os.Stdout
	if config.Logging.File != "" {
		logFile, err := os.OpenFile(config.Logging.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("无法打开日志文件 %s: %v，将仅使用标准输出", config.Logging.File, err)
		} else {
			// 使用MultiWriter同时输出到日志文件和标准输出
			logWriter = io.MultiWriter(os.Stdout, logFile)
		}
	}
	logger = log.New(logWriter, "", log.LstdFlags)

	// 收集环境信息 (调试模式)
	debugEnvironment()

	// 检查依赖
	if err := checkDependencies(); err != nil {
		logger.Fatalf("依赖检查失败: %v", err)
	}

	logger.Println("开始备份过程...")

	// 创建临时目录
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		logger.Fatalf("无法创建临时目录: %v", err)
	}

	// 如果启用了Docker备份，先暂停容器
	containersToResume := []string{}
	if config.Docker.Enabled {
		containersToResume = pauseContainers()
	}

	// 使用defer确保即使发生错误，容器也能被恢复
	defer func() {
		// 恢复暂停的容器
		if len(containersToResume) > 0 {
			resumeContainers(containersToResume)
		}
	}()

	// 执行本地备份
	backupMySQL()
	backupDocker()
	backupFiles()

	// 手动恢复容器，这样在本地备份完成后立即恢复，而不是等到整个备份过程结束
	if len(containersToResume) > 0 {
		resumeContainers(containersToResume)
		// 清空列表，防止defer中重复恢复
		containersToResume = nil
	}

	// 上传到 WebDAV
	uploadToWebDAV()

	// 清理旧备份
	cleanupOldBackups()

	logger.Println("备份过程完成")
}

func backupMySQL() {
	if !config.MySQL.Enabled {
		logger.Println("MySQL 备份已禁用，跳过")
		return
	}

	logger.Println("开始 MySQL 备份...")
	backupDir := filepath.Join(tempDir, "mysql")
	createBackupDir(backupDir)

	// 构建基础参数
	args := []string{
		"-u" + config.MySQL.User,
	}

	// 只有在密码非空时才添加密码参数
	if config.MySQL.Password != "" {
		args = append(args, "-p"+config.MySQL.Password)
	}

	// 添加主机和端口参数
	if config.MySQL.Host != "" {
		args = append(args, "-h"+config.MySQL.Host)
	}
	if config.MySQL.Port > 0 {
		args = append(args, fmt.Sprintf("-P%d", config.MySQL.Port))
		args = append(args, "--protocol=tcp")
	}

	if len(config.MySQL.Databases) == 1 && config.MySQL.Databases[0] == "all" {
		args = append(args, "--all-databases")
		outputFile := filepath.Join(backupDir, fmt.Sprintf("all_databases_%s.sql.gz", dateStr))
		if err := runPipedCommand(
			exec.Command("mysqldump", args...),
			exec.Command("gzip", "-c"),
			outputFile,
		); err != nil {
			logger.Printf("MySQL 备份失败: %v", err)
			return
		}
	} else {
		for _, db := range config.MySQL.Databases {
			dbArgs := append(args, db)
			outputFile := filepath.Join(backupDir, fmt.Sprintf("%s_%s.sql.gz", db, dateStr))
			if err := runPipedCommand(
				exec.Command("mysqldump", dbArgs...),
				exec.Command("gzip", "-c"),
				outputFile,
			); err != nil {
				logger.Printf("数据库 %s 备份失败: %v", db, err)
				continue
			}
		}
	}

	logger.Println("MySQL 备份完成")
}

// 执行管道命令，将第一个命令的输出通过管道传递给第二个命令
func runPipedCommand(cmd1 *exec.Cmd, cmd2 *exec.Cmd, outputFile string) error {
	// 记录命令执行（使用处理过的参数）
	sanitizedArgs1 := sanitizeArgsForLog(cmd1.Path, cmd1.Args[1:])
	sanitizedArgs2 := sanitizeArgsForLog(cmd2.Path, cmd2.Args[1:])
	logger.Printf("执行命令: %s %v | %s %v > %s",
		filepath.Base(cmd1.Path), sanitizedArgs1,
		filepath.Base(cmd2.Path), sanitizedArgs2,
		outputFile)

	// 创建管道
	r, w := io.Pipe()

	// 设置命令的输入输出
	cmd1.Stdout = w
	cmd2.Stdin = r

	// 打开输出文件
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()
	cmd2.Stdout = file

	// 捕获错误输出
	var stderr1, stderr2 bytes.Buffer
	cmd1.Stderr = &stderr1
	cmd2.Stderr = &stderr2

	// 启动命令
	if err := cmd1.Start(); err != nil {
		return fmt.Errorf("启动第一个命令失败: %w", err)
	}
	if err := cmd2.Start(); err != nil {
		return fmt.Errorf("启动第二个命令失败: %w", err)
	}

	// 等待命令完成
	go func() {
		cmd1.Wait()
		w.Close()
	}()

	if err := cmd2.Wait(); err != nil {
		err1 := stderr1.String()
		err2 := stderr2.String()
		if err1 != "" || err2 != "" {
			return fmt.Errorf("命令执行失败:\n%s\n%s", err1, err2)
		}
		return err
	}

	return nil
}

func backupDocker() {
	if !config.Docker.Enabled {
		logger.Println("Docker备份已禁用")
		return
	}

	backupDir := filepath.Join(tempDir, "docker")
	createBackupDir(backupDir)

	// 备份容器
	containers := getDockerContainers()
	for _, container := range containers {
		// 备份容器配置
		inspectFile := filepath.Join(backupDir, fmt.Sprintf("%s.json", container))
		if err := runCommand("docker", []string{"inspect", container}, inspectFile); err != nil {
			logger.Printf("%v", err)
		}

		// 导出容器
		exportFile := filepath.Join(backupDir, fmt.Sprintf("%s.tar", container))
		if err := runCommand("docker", []string{"export", "-o", exportFile, container}, ""); err != nil {
			logger.Printf("%v", err)
		}
	}

	// 备份docker-compose文件
	if config.Docker.BackupCompose {
		composeDir := filepath.Join(backupDir, "compose")
		createBackupDir(composeDir)

		for _, pattern := range config.Docker.ComposePaths {
			matches, _ := filepath.Glob(pattern)
			for _, file := range matches {
				dest := filepath.Join(composeDir, filepath.Base(file))
				if err := runCommand("cp", []string{file, dest}, ""); err != nil {
					logger.Printf("%v", err)
				}
			}
		}
	}

	// 备份数据卷
	if config.Docker.BackupVolumes {
		volumesDir := filepath.Join(backupDir, "volumes")
		createBackupDir(volumesDir)

		volumes := getDockerVolumes()
		for _, volume := range volumes {
			src := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volume)
			dest := filepath.Join(volumesDir, fmt.Sprintf("%s.tar.gz", volume))
			if err := runCommand("tar", []string{"-czf", dest, "-C", src, "."}, ""); err != nil {
				logger.Printf("%v", err)
			}
		}
	}
}

// 辅助函数
func getDockerContainers() []string {
	if config.Docker.Containers[0] == "all" {
		out, _ := exec.Command("docker", "ps", "-aq").Output()
		return strings.Fields(string(out))
	}
	return config.Docker.Containers
}

func getDockerVolumes() []string {
	out, _ := exec.Command("docker", "volume", "ls", "-q").Output()
	return strings.Fields(string(out))
}

func backupFiles() {
	if !config.Files.Enabled {
		logger.Println("文件备份已禁用，跳过")
		return
	}

	logger.Println("开始文件备份...")
	backupDir := filepath.Join(tempDir, "files")
	createBackupDir(backupDir)

	for _, path := range config.Files.Paths {
		parts := strings.Split(path, ":")
		if len(parts) != 2 {
			logger.Printf("无效的文件备份路径格式: %s", path)
			continue
		}

		srcPath := parts[0]
		backupName := parts[1]
		outputFile := filepath.Join(backupDir, fmt.Sprintf("%s_%s.tar.gz", backupName, dateStr))

		if err := runCommand("tar", []string{"-czf", outputFile, "-C", filepath.Dir(srcPath), filepath.Base(srcPath)}, ""); err != nil {
			logger.Fatalf("文件 %s 备份失败: %v", srcPath, err)
		}
	}

	logger.Println("文件备份完成")
}

func uploadToWebDAV() {
	logger.Println("开始上传到 WebDAV...")
	remotePath := fmt.Sprintf("%s:%s/%s", config.WebDAV.Remote, config.WebDAV.Path, dateStr)

	// 构建mkdir命令参数
	args := []string{"mkdir", remotePath}
	// 添加配置文件路径参数
	if config.WebDAV.RcloneConfig != "" {
		args = []string{"--config", config.WebDAV.RcloneConfig, "mkdir", remotePath}
		logger.Printf("使用自定义rclone配置: %s", config.WebDAV.RcloneConfig)
	}

	if err := runCommand("rclone", args, ""); err != nil {
		logger.Fatalf("无法创建 WebDAV 目录: %v", err)
	}

	// 构建copy命令参数
	copyArgs := []string{"copy", tempDir, remotePath, "--progress"}
	// 添加配置文件路径参数
	if config.WebDAV.RcloneConfig != "" {
		copyArgs = []string{"--config", config.WebDAV.RcloneConfig, "copy", tempDir, remotePath, "--progress"}
	}

	if err := runCommand("rclone", copyArgs, ""); err != nil {
		logger.Fatalf("上传到 WebDAV 失败: %v", err)
	}

	logger.Println("上传到 WebDAV 成功")
}

func cleanupOldBackups() {
	logger.Println("开始清理旧备份...")
	cutoffDate := time.Now().AddDate(0, 0, -config.WebDAV.RetentionDays).Format("20060102")

	remotePath := fmt.Sprintf("%s:%s", config.WebDAV.Remote, config.WebDAV.Path)

	// 构建lsd命令参数
	lsdArgs := []string{"lsd", remotePath}
	// 添加配置文件路径参数
	if config.WebDAV.RcloneConfig != "" {
		lsdArgs = []string{"--config", config.WebDAV.RcloneConfig, "lsd", remotePath}
	}

	out, err := exec.Command("rclone", lsdArgs...).Output()
	if err != nil {
		logger.Printf("无法获取 WebDAV 备份列表: %v", err)
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		folder := fields[4]
		if len(folder) < 8 {
			continue
		}

		folderDate := folder[:8]
		if folderDate < cutoffDate {
			logger.Printf("删除旧备份: %s", folder)
			remoteFolder := fmt.Sprintf("%s/%s", remotePath, folder)

			// 构建purge命令参数
			purgeArgs := []string{"purge", remoteFolder}
			// 添加配置文件路径参数
			if config.WebDAV.RcloneConfig != "" {
				purgeArgs = []string{"--config", config.WebDAV.RcloneConfig, "purge", remoteFolder}
			}

			if err := runCommand("rclone", purgeArgs, ""); err != nil {
				logger.Printf("删除备份 %s 失败: %v", folder, err)
			}
		}
	}

	// 清理本地临时文件
	if err := os.RemoveAll(tempDir); err != nil {
		logger.Printf("清理临时目录失败: %v", err)
	}

	logger.Println("清理完成")
}

// 用于日志记录的安全参数处理
func sanitizeArgsForLog(name string, args []string) []string {
	// 复制参数数组以避免修改原始数据
	sanitized := make([]string, len(args))
	copy(sanitized, args)

	// 根据命令类型处理敏感参数
	switch name {
	case "mysqldump":
		for i, arg := range sanitized {
			if strings.HasPrefix(arg, "-p") && len(arg) > 2 {
				sanitized[i] = "-p******"
			}
		}
	}
	return sanitized
}

// 统一命令执行函数
func runCommand(name string, args []string, outputFile string) error {
	// 记录命令执行（使用处理过的参数）
	sanitizedArgs := sanitizeArgsForLog(name, args)
	if outputFile != "" {
		logger.Printf("执行命令: %s %v > %s", name, sanitizedArgs, outputFile)
	} else {
		logger.Printf("执行命令: %s %v", name, sanitizedArgs)
	}

	cmd := exec.Command(name, args...)

	// 设置明确的工作目录
	cmd.Dir = os.TempDir()

	// 记录当前环境变量
	logger.Printf("环境HOME=%s", os.Getenv("HOME"))

	// 处理输出重定向
	var stdout bytes.Buffer
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("创建输出文件失败: %w", err)
		}
		defer file.Close()
		cmd.Stdout = io.MultiWriter(file, &stdout) // 同时捕获到文件和内存
	} else {
		cmd.Stdout = &stdout
	}

	// 捕获标准错误
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// 执行命令
	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	// 记录执行时间
	logger.Printf("命令执行时间: %v", duration)

	// 记录所有输出
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if stdoutStr != "" && len(stdoutStr) < 1000 {
		logger.Printf("命令标准输出: %s", stdoutStr)
	} else if stdoutStr != "" {
		logger.Printf("命令标准输出: [输出过长，已省略]")
	}

	if err != nil {
		if stderrStr != "" {
			logger.Printf("命令错误输出: %s", stderrStr)
			return fmt.Errorf("命令执行失败: %s %v → %s", name, sanitizedArgs, stderrStr)
		}
		return fmt.Errorf("命令执行失败: %s %v → %s", name, sanitizedArgs, err.Error())
	}

	return nil
}

func createBackupDir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		logger.Fatalf("创建目录失败: %s → %v", path, err)
	}
}
