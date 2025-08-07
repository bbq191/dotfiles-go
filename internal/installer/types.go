package installer

import (
	"context"
	"github.com/sirupsen/logrus"
)

// PackageManager 包管理器接口 - MVP设计
type PackageManager interface {
	// Name 返回包管理器名称
	Name() string
	
	// IsAvailable 检查包管理器是否可用
	IsAvailable() bool
	
	// Install 安装单个包
	Install(ctx context.Context, packageName string) error
	
	// IsInstalled 检查包是否已安装
	IsInstalled(packageName string) bool
	
	// Priority 返回优先级 (数值越低优先级越高)
	Priority() int
}

// InstallOptions 安装选项
type InstallOptions struct {
	Force      bool // 强制重新安装
	DryRun     bool // 仅显示将要执行的操作
	Verbose    bool // 详细输出
	Quiet      bool // 静默模式，不显示进度条
	Parallel   bool // 启用并行安装
	MaxWorkers int  // 最大并行工作数
}

// InstallResult 安装结果
type InstallResult struct {
	PackageName string
	Manager     string
	Success     bool
	Skipped     bool    // 是否跳过安装（包已存在）
	Error       error
	Duration    float64 // 安装耗时（秒）
}

// Installer 安装器核心
type Installer struct {
	managers []PackageManager
	logger   *logrus.Logger
}

// NewInstaller 创建新的安装器实例
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		managers: make([]PackageManager, 0),
		logger:   logger,
	}
}

// RegisterManager 注册包管理器
func (i *Installer) RegisterManager(manager PackageManager) {
	i.managers = append(i.managers, manager)
	i.logger.Debugf("注册包管理器: %s (优先级: %d)", manager.Name(), manager.Priority())
}

// GetAvailableManagers 获取可用的包管理器列表
func (i *Installer) GetAvailableManagers() []PackageManager {
	available := make([]PackageManager, 0)
	for _, manager := range i.managers {
		if manager.IsAvailable() {
			available = append(available, manager)
		}
	}
	return available
}

// SelectManager 为包选择最合适的管理器
func (i *Installer) SelectManager() PackageManager {
	available := i.GetAvailableManagers()
	if len(available) == 0 {
		return nil
	}
	
	// 选择优先级最高（数值最小）的管理器
	best := available[0]
	for _, manager := range available[1:] {
		if manager.Priority() < best.Priority() {
			best = manager
		}
	}
	
	return best
}