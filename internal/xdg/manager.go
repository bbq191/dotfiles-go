package xdg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetXDGPath 获取指定类型的XDG目录路径
func (m *Manager) GetXDGPath(dirType XDGDirectory) (string, error) {
	var envVar, defaultPath string
	
	switch dirType {
	case ConfigHome:
		envVar = "XDG_CONFIG_HOME"
		defaultPath = m.getDefaultConfigHome()
	case DataHome:
		envVar = "XDG_DATA_HOME"
		defaultPath = m.getDefaultDataHome()
	case StateHome:
		envVar = "XDG_STATE_HOME"
		defaultPath = m.getDefaultStateHome()
	case CacheHome:
		envVar = "XDG_CACHE_HOME"
		defaultPath = m.getDefaultCacheHome()
	case RuntimeDir:
		envVar = "XDG_RUNTIME_DIR"
		defaultPath = m.getDefaultRuntimeDir()
	case UserBin:
		// 用户二进制目录，不是标准XDG但常用
		return m.getDefaultUserBin(), nil
	default:
		return "", fmt.Errorf("未支持的XDG目录类型: %v", dirType)
	}
	
	// 优先使用环境变量
	if path := os.Getenv(envVar); path != "" {
		return m.expandPath(path), nil
	}
	
	return m.expandPath(defaultPath), nil
}

// EnsureDirectories 确保所有XDG目录存在
func (m *Manager) EnsureDirectories() error {
	directories := []XDGDirectory{
		ConfigHome, DataHome, StateHome, CacheHome, UserBin,
	}
	
	for _, dirType := range directories {
		path, err := m.GetXDGPath(dirType)
		if err != nil {
			m.logger.Warnf("获取%s路径失败: %v", dirType.String(), err)
			continue
		}
		
		if err := os.MkdirAll(path, 0755); err != nil {
			m.logger.Errorf("创建目录失败 %s: %v", path, err)
			return fmt.Errorf("创建XDG目录失败 %s: %w", path, err)
		}
		
		m.logger.Debugf("✅ 确保目录存在: %s", path)
	}
	
	return nil
}

// ValidateDirectories 验证XDG目录的权限和可访问性
func (m *Manager) ValidateDirectories() error {
	directories := []XDGDirectory{
		ConfigHome, DataHome, StateHome, CacheHome, UserBin,
	}
	
	for _, dirType := range directories {
		path, err := m.GetXDGPath(dirType)
		if err != nil {
			continue
		}
		
		// 检查目录是否存在
		if _, err := os.Stat(path); os.IsNotExist(err) {
			m.logger.Warnf("XDG目录不存在: %s", path)
			continue
		}
		
		// 检查读写权限
		if !m.isDirectoryWritable(path) {
			return fmt.Errorf("XDG目录不可写: %s", path)
		}
		
		m.logger.Debugf("✅ XDG目录验证通过: %s", path)
	}
	
	return nil
}

// CheckCompliance 检查当前系统的XDG合规性
func (m *Manager) CheckCompliance() ([]ComplianceIssue, error) {
	var issues []ComplianceIssue
	
	// 检查基础XDG环境变量
	xdgEnvVars := map[string]XDGDirectory{
		"XDG_CONFIG_HOME": ConfigHome,
		"XDG_DATA_HOME":   DataHome,
		"XDG_STATE_HOME":  StateHome,
		"XDG_CACHE_HOME":  CacheHome,
	}
	
	for envVar, dirType := range xdgEnvVars {
		if os.Getenv(envVar) == "" {
			defaultPath, _ := m.GetXDGPath(dirType)
			issues = append(issues, ComplianceIssue{
				Application:     "system",
				IssueType:       "missing_env_var",
				Description:     fmt.Sprintf("XDG环境变量 %s 未设置", envVar),
				CurrentPath:     "",
				RecommendedPath: defaultPath,
				Severity:        "medium",
				AutoFixable:     true,
			})
		}
	}
	
	// 检查常见应用的非XDG路径
	issues = append(issues, m.checkCommonApplications()...)
	
	return issues, nil
}

// FixComplianceIssue 修复合规性问题
func (m *Manager) FixComplianceIssue(issue ComplianceIssue) error {
	if !issue.AutoFixable {
		return fmt.Errorf("问题不可自动修复: %s", issue.Description)
	}
	
	switch issue.IssueType {
	case "missing_env_var":
		// 这里只能提示用户手动设置环境变量
		// 因为程序设置的环境变量不会持久化
		m.logger.Infof("请在shell配置中设置: export %s=%s", 
			issue.Application, issue.RecommendedPath)
		return nil
		
	case "non_xdg_path":
		// 移动文件到XDG路径
		return m.moveToXDGPath(issue.CurrentPath, issue.RecommendedPath)
		
	default:
		return fmt.Errorf("未知的问题类型: %s", issue.IssueType)
	}
}

// 平台特定的默认路径实现
func (m *Manager) getDefaultConfigHome() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"))
	default: // linux, darwin
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config")
	}
}

func (m *Manager) getDefaultDataHome() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"))
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share")
	}
}

func (m *Manager) getDefaultStateHome() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "State")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "state")
	}
}

func (m *Manager) getDefaultCacheHome() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Temp")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".cache")
	}
}

func (m *Manager) getDefaultRuntimeDir() string {
	switch runtime.GOOS {
	case "linux":
		uid := os.Getuid()
		return fmt.Sprintf("/run/user/%d", uid)
	default:
		return os.TempDir()
	}
}

func (m *Manager) getDefaultUserBin() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "bin")
	}
}

// 辅助方法
func (m *Manager) isDirectoryWritable(path string) bool {
	testFile := filepath.Join(path, ".dotfiles_test")
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}

func (m *Manager) checkCommonApplications() []ComplianceIssue {
	var issues []ComplianceIssue
	home, _ := os.UserHomeDir()
	
	// 检查常见的非XDG路径 - 映射到实际的应用配置名称
	commonPaths := map[string]struct{
		AppName     string
		Description string
		XDGSubdir   string
	}{
		".zshrc": {
			AppName:     "zsh",
			Description: "zsh配置应移动到 $XDG_CONFIG_HOME/zsh/",
			XDGSubdir:   "zsh",
		},
		".bashrc": {
			AppName:     "bash", 
			Description: "bash配置应移动到 $XDG_CONFIG_HOME/bash/",
			XDGSubdir:   "bash",
		},
		".vimrc": {
			AppName:     "vim",
			Description: "vim配置应移动到 $XDG_CONFIG_HOME/vim/",
			XDGSubdir:   "vim",
		},
		".gitconfig": {
			AppName:     "git",
			Description: "git配置应移动到 $XDG_CONFIG_HOME/git/",
			XDGSubdir:   "git",
		},
		".ssh/config": {
			AppName:     "ssh",
			Description: "SSH配置建议移动到 $XDG_CONFIG_HOME/ssh/",
			XDGSubdir:   "ssh",
		},
	}
	
	for relativePath, config := range commonPaths {
		fullPath := filepath.Join(home, relativePath)
		if _, err := os.Stat(fullPath); err == nil {
			configHome, _ := m.GetXDGPath(ConfigHome)
			recommendedPath := filepath.Join(configHome, config.XDGSubdir)
			
			issues = append(issues, ComplianceIssue{
				Application:     config.AppName,
				IssueType:       "non_xdg_path",
				Description:     config.Description,
				CurrentPath:     fullPath,
				RecommendedPath: recommendedPath,
				Severity:        "low",
				AutoFixable:     true,
			})
		}
	}
	
	return issues
}

func (m *Manager) moveToXDGPath(sourcePath, targetPath string) error {
	// 确保目标目录存在
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}
	
	// 检查目标文件是否已存在
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("目标文件已存在: %s", targetPath)
	}
	
	// 移动文件
	if err := os.Rename(sourcePath, targetPath); err != nil {
		return fmt.Errorf("移动文件失败: %w", err)
	}
	
	m.logger.Infof("✅ 文件已移动: %s -> %s", sourcePath, targetPath)
	return nil
}