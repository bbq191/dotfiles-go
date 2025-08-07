package installer

import (
	"context"
	"testing"
	
	"github.com/sirupsen/logrus"
)

// MockPackageManager 用于测试的模拟包管理器
type MockPackageManager struct {
	name           string
	available      bool
	priority       int
	installedPkgs  map[string]bool
	installError   error
}

func NewMockPackageManager(name string, priority int) *MockPackageManager {
	return &MockPackageManager{
		name:          name,
		available:     true,
		priority:      priority,
		installedPkgs: make(map[string]bool),
		installError:  nil,
	}
}

func (m *MockPackageManager) Name() string {
	return m.name
}

func (m *MockPackageManager) IsAvailable() bool {
	return m.available
}

func (m *MockPackageManager) Install(ctx context.Context, packageName string) error {
	if m.installError != nil {
		return m.installError
	}
	m.installedPkgs[packageName] = true
	return nil
}

func (m *MockPackageManager) IsInstalled(packageName string) bool {
	return m.installedPkgs[packageName]
}

func (m *MockPackageManager) Priority() int {
	return m.priority
}

// 设置包为已安装状态（测试辅助方法）
func (m *MockPackageManager) SetInstalled(packageName string, installed bool) {
	m.installedPkgs[packageName] = installed
}

// 设置安装错误（测试辅助方法）
func (m *MockPackageManager) SetInstallError(err error) {
	m.installError = err
}

// 设置可用状态（测试辅助方法）
func (m *MockPackageManager) SetAvailable(available bool) {
	m.available = available
}

// TestNewInstaller 测试安装器创建
func TestNewInstaller(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	if installer == nil {
		t.Fatal("NewInstaller 应该返回非空实例")
	}
	
	if installer.logger != logger {
		t.Error("安装器应该使用提供的logger")
	}
	
	if len(installer.managers) != 0 {
		t.Error("新创建的安装器应该没有注册的管理器")
	}
}

// TestRegisterManager 测试管理器注册
func TestRegisterManager(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	mockManager := NewMockPackageManager("test-manager", 1)
	installer.RegisterManager(mockManager)
	
	if len(installer.managers) != 1 {
		t.Errorf("期望注册 1 个管理器，实际注册了 %d 个", len(installer.managers))
	}
	
	if installer.managers[0] != mockManager {
		t.Error("注册的管理器应该是提供的管理器实例")
	}
}

// TestGetAvailableManagers 测试获取可用管理器
func TestGetAvailableManagers(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	// 注册一个可用的管理器
	availableManager := NewMockPackageManager("available", 1)
	installer.RegisterManager(availableManager)
	
	// 注册一个不可用的管理器
	unavailableManager := NewMockPackageManager("unavailable", 2)
	unavailableManager.SetAvailable(false)
	installer.RegisterManager(unavailableManager)
	
	available := installer.GetAvailableManagers()
	
	if len(available) != 1 {
		t.Errorf("期望 1 个可用管理器，实际获得 %d 个", len(available))
	}
	
	if available[0].Name() != "available" {
		t.Errorf("期望可用管理器名称为 'available'，实际为 '%s'", available[0].Name())
	}
}

// TestSelectManager 测试管理器选择
func TestSelectManager(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	// 注册高优先级管理器 (优先级数值低)
	highPriority := NewMockPackageManager("high-priority", 1)
	installer.RegisterManager(highPriority)
	
	// 注册低优先级管理器 (优先级数值高)
	lowPriority := NewMockPackageManager("low-priority", 3)
	installer.RegisterManager(lowPriority)
	
	selected := installer.SelectManager()
	
	if selected == nil {
		t.Fatal("SelectManager 应该返回一个管理器")
	}
	
	if selected.Name() != "high-priority" {
		t.Errorf("期望选择高优先级管理器 'high-priority'，实际选择了 '%s'", selected.Name())
	}
}

// TestSelectManager_NoAvailable 测试没有可用管理器的情况
func TestSelectManager_NoAvailable(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	// 注册不可用的管理器
	unavailable := NewMockPackageManager("unavailable", 1)
	unavailable.SetAvailable(false)
	installer.RegisterManager(unavailable)
	
	selected := installer.SelectManager()
	
	if selected != nil {
		t.Error("当没有可用管理器时，SelectManager 应该返回 nil")
	}
}

// TestInstallPackage_Success 测试成功安装包
func TestInstallPackage_Success(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	mockManager := NewMockPackageManager("test", 1)
	installer.RegisterManager(mockManager)
	
	ctx := context.Background()
	opts := InstallOptions{}
	
	result, err := installer.InstallPackage(ctx, "test-package", opts)
	
	if err != nil {
		t.Errorf("安装应该成功，但返回错误: %v", err)
	}
	
	if result == nil {
		t.Fatal("InstallPackage 应该返回结果")
	}
	
	if !result.Success {
		t.Error("安装结果应该标记为成功")
	}
	
	if result.PackageName != "test-package" {
		t.Errorf("期望包名为 'test-package'，实际为 '%s'", result.PackageName)
	}
	
	if result.Manager != "test" {
		t.Errorf("期望管理器为 'test'，实际为 '%s'", result.Manager)
	}
}

// TestInstallPackage_AlreadyInstalled 测试跳过已安装包
func TestInstallPackage_AlreadyInstalled(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	mockManager := NewMockPackageManager("test", 1)
	mockManager.SetInstalled("existing-package", true)
	installer.RegisterManager(mockManager)
	
	ctx := context.Background()
	opts := InstallOptions{Force: false}
	
	result, err := installer.InstallPackage(ctx, "existing-package", opts)
	
	if err != nil {
		t.Errorf("跳过已安装包应该成功，但返回错误: %v", err)
	}
	
	if !result.Success {
		t.Error("跳过已安装包应该标记为成功")
	}
}

// TestInstallPackage_DryRun 测试预览模式
func TestInstallPackage_DryRun(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	mockManager := NewMockPackageManager("test", 1)
	installer.RegisterManager(mockManager)
	
	ctx := context.Background()
	opts := InstallOptions{DryRun: true}
	
	result, err := installer.InstallPackage(ctx, "test-package", opts)
	
	if err != nil {
		t.Errorf("预览模式应该成功，但返回错误: %v", err)
	}
	
	if !result.Success {
		t.Error("预览模式应该标记为成功")
	}
	
	// 确保实际没有安装
	if mockManager.IsInstalled("test-package") {
		t.Error("预览模式不应该实际安装包")
	}
}

// TestInstallPackages_Multiple 测试批量安装
func TestInstallPackages_Multiple(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	mockManager := NewMockPackageManager("test", 1)
	installer.RegisterManager(mockManager)
	
	ctx := context.Background()
	opts := InstallOptions{}
	packages := []string{"pkg1", "pkg2", "pkg3"}
	
	results, err := installer.InstallPackages(ctx, packages, opts)
	
	if err != nil {
		t.Errorf("批量安装应该成功，但返回错误: %v", err)
	}
	
	if len(results) != 3 {
		t.Errorf("期望 3 个结果，实际获得 %d 个", len(results))
	}
	
	for i, result := range results {
		if !result.Success {
			t.Errorf("包 %d 安装应该成功", i)
		}
		
		if result.PackageName != packages[i] {
			t.Errorf("结果 %d 的包名应该是 '%s'，实际为 '%s'", 
				i, packages[i], result.PackageName)
		}
	}
}