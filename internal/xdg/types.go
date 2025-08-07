package xdg

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// XDGDirectory XDG目录类型枚举
type XDGDirectory int

const (
	ConfigHome XDGDirectory = iota
	DataHome
	StateHome
	CacheHome
	RuntimeDir
	UserBin
)

// String 返回XDG目录类型的字符串表示
func (d XDGDirectory) String() string {
	switch d {
	case ConfigHome:
		return "config"
	case DataHome:
		return "data"
	case StateHome:
		return "state"
	case CacheHome:
		return "cache"
	case RuntimeDir:
		return "runtime"
	case UserBin:
		return "bin"
	default:
		return "unknown"
	}
}

// DirectorySpec XDG目录规范定义
type DirectorySpec struct {
	Type         XDGDirectory `json:"type"`
	EnvVar       string       `json:"env_var"`
	DefaultPath  string       `json:"default_path"`
	Description  string       `json:"description"`
	Required     bool         `json:"required"`
	Permissions  os.FileMode  `json:"permissions"`
}

// ApplicationConfig 应用的XDG配置
type ApplicationConfig struct {
	Name         string            `json:"name"`
	Enabled      bool              `json:"enabled"`
	ConfigFiles  map[string]string `json:"config_files"`  // 原路径 -> XDG路径
	DataFiles    map[string]string `json:"data_files"`    // 原路径 -> XDG路径
	CacheFiles   map[string]string `json:"cache_files"`   // 原路径 -> XDG路径
	StateFiles   map[string]string `json:"state_files"`   // 原路径 -> XDG路径
	EnvVars      map[string]string `json:"env_vars"`      // 环境变量设置
	PostMigrate  []string          `json:"post_migrate"`  // 迁移后执行的命令
}

// MigrationTask 迁移任务
type MigrationTask struct {
	Application  string    `json:"application"`
	SourcePath   string    `json:"source_path"`
	TargetPath   string    `json:"target_path"`
	Type         string    `json:"type"`         // "file", "directory", "symlink"
	Action       string    `json:"action"`       // "move", "copy", "symlink"
	Backup       bool      `json:"backup"`
	Status       string    `json:"status"`       // "pending", "completed", "failed", "skipped"
	Error        error     `json:"error,omitempty"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
}

// ComplianceIssue 合规性问题
type ComplianceIssue struct {
	Application  string `json:"application"`
	IssueType    string `json:"issue_type"`    // "non_xdg_path", "missing_env_var", "incorrect_permissions"
	Description  string `json:"description"`
	CurrentPath  string `json:"current_path"`
	RecommendedPath string `json:"recommended_path"`
	Severity     string `json:"severity"`      // "low", "medium", "high"
	AutoFixable  bool   `json:"auto_fixable"`
}

// XDGManager XDG管理器接口
type XDGManager interface {
	// 目录管理
	GetXDGPath(dirType XDGDirectory) (string, error)
	EnsureDirectories() error
	ValidateDirectories() error
	
	// 合规性检查
	CheckCompliance() ([]ComplianceIssue, error)
	FixComplianceIssue(issue ComplianceIssue) error
	
	// 迁移功能
	PlanMigration(applications []string) ([]MigrationTask, error)
	ExecuteMigration(tasks []MigrationTask, options MigrationOptions) error
	RollbackMigration(backupDir string) error
	
	// 配置管理
	LoadApplicationConfigs() (map[string]ApplicationConfig, error)
	GetApplicationConfig(appName string) (*ApplicationConfig, error)
}

// MigrationOptions 迁移选项
type MigrationOptions struct {
	Force         bool   `json:"force"`           // 强制迁移，覆盖现有文件
	Backup        bool   `json:"backup"`          // 创建备份
	BackupDir     string `json:"backup_dir"`      // 备份目录
	DryRun        bool   `json:"dry_run"`         // 预演模式，不实际执行
	Interactive   bool   `json:"interactive"`     // 交互式确认
	Parallel      bool   `json:"parallel"`        // 并行执行
	MaxWorkers    int    `json:"max_workers"`     // 最大工作协程数
	IgnoreErrors  bool   `json:"ignore_errors"`   // 忽略错误继续执行
	Verbose       bool   `json:"verbose"`         // 详细输出
}

// XDGConfig XDG配置结构
type XDGConfig struct {
	Enabled      bool                         `json:"enabled"`
	Directories  map[string]DirectorySpec     `json:"directories"`
	Applications map[string]ApplicationConfig `json:"applications"`
}

// Manager XDG管理器实现
type Manager struct {
	config   *XDGConfig
	logger   *logrus.Logger
	platform string // linux, windows, macos
}

// NewManager 创建新的XDG管理器
func NewManager(logger *logrus.Logger, platform string) *Manager {
	return &Manager{
		logger:   logger,
		platform: platform,
	}
}

// 内部辅助方法
func (m *Manager) expandPath(path string) string {
	// 展开环境变量和用户目录
	expanded := os.ExpandEnv(path)
	if expanded[0] == '~' {
		home, _ := os.UserHomeDir()
		expanded = filepath.Join(home, expanded[1:])
	}
	return expanded
}

// 平台相关路径获取
func (m *Manager) getPlatformSpecificPath(paths map[string]string) string {
	if path, exists := paths[m.platform]; exists {
		return m.expandPath(path)
	}
	if path, exists := paths["linux"]; exists { // fallback
		return m.expandPath(path)
	}
	return ""
}