package xdg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// PlanMigration 计划迁移任务
func (m *Manager) PlanMigration(applications []string) ([]MigrationTask, error) {
	var tasks []MigrationTask
	
	// 加载应用配置
	appConfigs, err := m.LoadApplicationConfigs()
	if err != nil {
		return nil, fmt.Errorf("加载应用配置失败: %w", err)
	}
	
	// 为每个应用生成迁移任务
	for _, appName := range applications {
		config, exists := appConfigs[appName]
		if !exists {
			m.logger.Warnf("未找到应用 %s 的配置", appName)
			continue
		}
		
		if !config.Enabled {
			m.logger.Debugf("应用 %s 已禁用迁移", appName)
			continue
		}
		
		// 生成配置文件迁移任务
		for source, target := range config.ConfigFiles {
			task, err := m.createMigrationTask(appName, source, target, "config")
			if err != nil {
				m.logger.Warnf("创建配置迁移任务失败 %s: %v", source, err)
				continue
			}
			if task != nil {
				tasks = append(tasks, *task)
			}
		}
		
		// 生成数据文件迁移任务
		for source, target := range config.DataFiles {
			task, err := m.createMigrationTask(appName, source, target, "data")
			if err != nil {
				m.logger.Warnf("创建数据迁移任务失败 %s: %v", source, err)
				continue
			}
			if task != nil {
				tasks = append(tasks, *task)
			}
		}
		
		// 生成缓存文件迁移任务
		for source, target := range config.CacheFiles {
			task, err := m.createMigrationTask(appName, source, target, "cache")
			if err != nil {
				m.logger.Warnf("创建缓存迁移任务失败 %s: %v", source, err)
				continue
			}
			if task != nil {
				tasks = append(tasks, *task)
			}
		}
		
		// 生成状态文件迁移任务
		for source, target := range config.StateFiles {
			task, err := m.createMigrationTask(appName, source, target, "state")
			if err != nil {
				m.logger.Warnf("创建状态迁移任务失败 %s: %v", source, err)
				continue
			}
			if task != nil {
				tasks = append(tasks, *task)
			}
		}
	}
	
	m.logger.Infof("📋 已规划 %d 个迁移任务", len(tasks))
	return tasks, nil
}

// ExecuteMigration 执行迁移任务
func (m *Manager) ExecuteMigration(tasks []MigrationTask, options MigrationOptions) error {
	if len(tasks) == 0 {
		m.logger.Info("🎯 没有需要迁移的任务")
		return nil
	}
	
	// 创建备份目录
	var backupDir string
	if options.Backup {
		var err error
		backupDir, err = m.createBackupDir(options.BackupDir)
		if err != nil {
			return fmt.Errorf("创建备份目录失败: %w", err)
		}
		m.logger.Infof("📦 备份目录: %s", backupDir)
	}
	
	if options.DryRun {
		m.logger.Info("🔍 预演模式 - 不会实际执行迁移")
		return m.dryRunMigration(tasks, options)
	}
	
	// 并行或串行执行
	if options.Parallel && len(tasks) > 1 {
		return m.executeParallelMigration(tasks, options, backupDir)
	} else {
		return m.executeSequentialMigration(tasks, options, backupDir)
	}
}

// RollbackMigration 回滚迁移
func (m *Manager) RollbackMigration(backupDir string) error {
	if backupDir == "" {
		return fmt.Errorf("备份目录路径为空")
	}
	
	// 检查备份目录是否存在
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("备份目录不存在: %s", backupDir)
	}
	
	// 读取备份元数据
	metadataPath := filepath.Join(backupDir, "migration_metadata.json")
	metadata, err := m.loadBackupMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("加载备份元数据失败: %w", err)
	}
	
	m.logger.Infof("🔄 开始回滚迁移，共 %d 个任务", len(metadata.Tasks))
	
	// 反向执行迁移任务
	successCount := 0
	for i := len(metadata.Tasks) - 1; i >= 0; i-- {
		task := metadata.Tasks[i]
		if err := m.rollbackTask(task, backupDir); err != nil {
			m.logger.Errorf("回滚任务失败 %s: %v", task.SourcePath, err)
			continue
		}
		successCount++
	}
	
	m.logger.Infof("✅ 回滚完成，成功 %d 个，总计 %d 个", successCount, len(metadata.Tasks))
	return nil
}

// LoadApplicationConfigs 加载应用配置
func (m *Manager) LoadApplicationConfigs() (map[string]ApplicationConfig, error) {
	// 这里应该从配置文件加载，目前先返回一些常见应用的硬编码配置
	configs := make(map[string]ApplicationConfig)
	
	// Zsh配置
	configHome, _ := m.GetXDGPath(ConfigHome)
	stateHome, _ := m.GetXDGPath(StateHome)
	
	configs["zsh"] = ApplicationConfig{
		Name:    "zsh",
		Enabled: true,
		ConfigFiles: map[string]string{
			"~/.zshrc":     filepath.Join(configHome, "zsh", ".zshrc"),
			"~/.zprofile":  filepath.Join(configHome, "zsh", ".zprofile"),
			"~/.zsh_history": filepath.Join(stateHome, "zsh", "history"),
		},
		EnvVars: map[string]string{
			"ZDOTDIR": filepath.Join(configHome, "zsh"),
		},
	}
	
	// Git配置
	configs["git"] = ApplicationConfig{
		Name:    "git",
		Enabled: true,
		ConfigFiles: map[string]string{
			"~/.gitconfig": filepath.Join(configHome, "git", "config"),
		},
	}
	
	// Vim配置
	dataHome, _ := m.GetXDGPath(DataHome)
	cacheHome, _ := m.GetXDGPath(CacheHome)
	
	configs["vim"] = ApplicationConfig{
		Name:    "vim",
		Enabled: true,
		ConfigFiles: map[string]string{
			"~/.vimrc": filepath.Join(configHome, "vim", "vimrc"),
		},
		DataFiles: map[string]string{
			"~/.vim": filepath.Join(dataHome, "vim"),
		},
		CacheFiles: map[string]string{
			"~/.vim/swap": filepath.Join(cacheHome, "vim", "swap"),
		},
		StateFiles: map[string]string{
			"~/.viminfo": filepath.Join(stateHome, "vim", "viminfo"),
		},
	}
	
	return configs, nil
}

// GetApplicationConfig 获取特定应用配置
func (m *Manager) GetApplicationConfig(appName string) (*ApplicationConfig, error) {
	configs, err := m.LoadApplicationConfigs()
	if err != nil {
		return nil, err
	}
	
	config, exists := configs[appName]
	if !exists {
		return nil, fmt.Errorf("未找到应用配置: %s", appName)
	}
	
	return &config, nil
}

// 内部实现方法
func (m *Manager) createMigrationTask(appName, source, target, taskType string) (*MigrationTask, error) {
	// 展开路径中的环境变量和波浪号
	sourcePath := m.expandPath(source)
	targetPath := m.expandPath(target)
	
	// 检查源文件是否存在
	sourceInfo, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		// 源文件不存在，跳过
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("检查源文件失败: %w", err)
	}
	
	// 检查目标文件是否已存在
	if _, err := os.Stat(targetPath); err == nil {
		m.logger.Debugf("目标文件已存在，跳过: %s", targetPath)
		return &MigrationTask{
			Application: appName,
			SourcePath:  sourcePath,
			TargetPath:  targetPath,
			Type:        getFileType(sourceInfo),
			Action:      "skip",
			Status:      "skipped",
		}, nil
	}
	
	return &MigrationTask{
		Application: appName,
		SourcePath:  sourcePath,
		TargetPath:  targetPath,
		Type:        getFileType(sourceInfo),
		Action:      "move",
		Backup:      true,
		Status:      "pending",
	}, nil
}

func (m *Manager) executeSequentialMigration(tasks []MigrationTask, options MigrationOptions, backupDir string) error {
	successCount := 0
	
	for i, task := range tasks {
		m.logger.Infof("🔄 执行迁移任务 [%d/%d]: %s", i+1, len(tasks), task.Application)
		
		if err := m.executeSingleTask(&tasks[i], options, backupDir); err != nil {
			if !options.IgnoreErrors {
				return fmt.Errorf("迁移任务失败: %w", err)
			}
			m.logger.Errorf("迁移任务失败（已忽略）: %v", err)
			continue
		}
		
		successCount++
	}
	
	m.logger.Infof("✅ 迁移完成，成功 %d 个，总计 %d 个", successCount, len(tasks))
	return nil
}

func (m *Manager) executeParallelMigration(tasks []MigrationTask, options MigrationOptions, backupDir string) error {
	workers := options.MaxWorkers
	if workers <= 0 {
		workers = 4 // 默认4个工作协程
	}
	
	g := &errgroup.Group{}
	g.SetLimit(workers)
	
	var mu sync.Mutex
	successCount := 0
	
	for i := range tasks {
		task := &tasks[i]
		g.Go(func() error {
			if err := m.executeSingleTask(task, options, backupDir); err != nil {
				if !options.IgnoreErrors {
					return err
				}
				m.logger.Errorf("迁移任务失败（已忽略）: %v", err)
				return nil
			}
			
			mu.Lock()
			successCount++
			mu.Unlock()
			return nil
		})
	}
	
	if err := g.Wait(); err != nil {
		return fmt.Errorf("并行迁移失败: %w", err)
	}
	
	m.logger.Infof("✅ 并行迁移完成，成功 %d 个，总计 %d 个", successCount, len(tasks))
	return nil
}

func (m *Manager) executeSingleTask(task *MigrationTask, options MigrationOptions, backupDir string) error {
	if task.Status == "skipped" {
		return nil
	}
	
	task.Status = "running"
	
	// 确保目标目录存在
	targetDir := filepath.Dir(task.TargetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		task.Status = "failed"
		task.Error = err
		return fmt.Errorf("创建目标目录失败: %w", err)
	}
	
	// 创建备份
	if options.Backup && backupDir != "" {
		if err := m.createBackup(task.SourcePath, backupDir); err != nil {
			m.logger.Warnf("创建备份失败: %v", err)
		}
	}
	
	// 执行迁移
	switch task.Action {
	case "move":
		err := os.Rename(task.SourcePath, task.TargetPath)
		if err != nil {
			task.Status = "failed"
			task.Error = err
			return fmt.Errorf("移动文件失败: %w", err)
		}
	case "copy":
		err := m.copyFile(task.SourcePath, task.TargetPath)
		if err != nil {
			task.Status = "failed"
			task.Error = err
			return fmt.Errorf("复制文件失败: %w", err)
		}
	}
	
	task.Status = "completed"
	task.CompletedAt = time.Now()
	
	m.logger.Infof("✅ 迁移完成: %s -> %s", task.SourcePath, task.TargetPath)
	return nil
}

func (m *Manager) dryRunMigration(tasks []MigrationTask, options MigrationOptions) error {
	m.logger.Info("📋 预演模式迁移计划:")
	
	for i, task := range tasks {
		action := "移动"
		if task.Action == "copy" {
			action = "复制"
		}
		
		m.logger.Infof("[%d] %s %s: %s -> %s", 
			i+1, action, task.Application, task.SourcePath, task.TargetPath)
	}
	
	return nil
}

// 辅助函数
func getFileType(info os.FileInfo) string {
	if info.IsDir() {
		return "directory"
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "symlink"
	}
	return "file"
}

func (m *Manager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (m *Manager) createBackupDir(customDir string) (string, error) {
	var backupDir string
	if customDir != "" {
		backupDir = customDir
	} else {
		dataHome, _ := m.GetXDGPath(DataHome)
		timestamp := time.Now().Format("20060102-150405")
		backupDir = filepath.Join(dataHome, "dotfiles", "xdg-backup", timestamp)
	}
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	
	return backupDir, nil
}

func (m *Manager) createBackup(sourcePath, backupDir string) error {
	// 计算相对于家目录的路径作为备份路径
	home, _ := os.UserHomeDir()
	relPath, _ := filepath.Rel(home, sourcePath)
	backupPath := filepath.Join(backupDir, relPath)
	
	// 确保备份目录存在
	backupParentDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupParentDir, 0755); err != nil {
		return err
	}
	
	return m.copyFile(sourcePath, backupPath)
}

// 备份元数据结构
type BackupMetadata struct {
	Timestamp time.Time       `json:"timestamp"`
	Tasks     []MigrationTask `json:"tasks"`
}

func (m *Manager) loadBackupMetadata(path string) (*BackupMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var metadata BackupMetadata
	err = json.Unmarshal(data, &metadata)
	return &metadata, err
}

func (m *Manager) rollbackTask(task MigrationTask, backupDir string) error {
	home, _ := os.UserHomeDir()
	relPath, _ := filepath.Rel(home, task.SourcePath)
	backupPath := filepath.Join(backupDir, relPath)
	
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}
	
	// 删除当前XDG路径的文件（如果存在）
	if _, err := os.Stat(task.TargetPath); err == nil {
		if err := os.RemoveAll(task.TargetPath); err != nil {
			return fmt.Errorf("删除当前文件失败: %w", err)
		}
	}
	
	// 确保原目录存在
	sourceDir := filepath.Dir(task.SourcePath)
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		return fmt.Errorf("创建原目录失败: %w", err)
	}
	
	// 恢复备份文件
	return m.copyFile(backupPath, task.SourcePath)
}