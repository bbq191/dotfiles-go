package installer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// MockParallelManager 支持并行的模拟包管理器
type MockParallelManager struct {
	*MockPackageManager
	installDelay time.Duration
}

func NewMockParallelManager(name string, priority int) *MockParallelManager {
	return &MockParallelManager{
		MockPackageManager: NewMockPackageManager(name, priority),
		installDelay:       50 * time.Millisecond, // 模拟安装延迟
	}
}

func (m *MockParallelManager) Install(ctx context.Context, packageName string) error {
	// 模拟安装时间
	select {
	case <-time.After(m.installDelay):
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// 调用父类方法
	return m.MockPackageManager.Install(ctx, packageName)
}

// TestNewParallelInstaller 测试并行安装器创建
func TestNewParallelInstaller(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	parallelInst := NewParallelInstaller(installer, 4)
	
	if parallelInst == nil {
		t.Fatal("NewParallelInstaller 应该返回非空实例")
	}
	
	if parallelInst.maxWorkers != 4 {
		t.Errorf("期望 maxWorkers 为 4，实际为 %d", parallelInst.maxWorkers)
	}
	
	if cap(parallelInst.semaphore) != 4 {
		t.Errorf("信号量容量应该为 4，实际为 %d", cap(parallelInst.semaphore))
	}
}

// TestNewParallelInstaller_DefaultWorkers 测试默认工作协程数
func TestNewParallelInstaller_DefaultWorkers(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	
	parallelInst := NewParallelInstaller(installer, 0) // 使用默认值
	
	if parallelInst.maxWorkers <= 0 {
		t.Error("默认工作协程数应该大于 0")
	}
}

// TestGetOptimalWorkerCount 测试最佳工作协程数计算
func TestGetOptimalWorkerCount(t *testing.T) {
	tests := []struct {
		packageCount int
		expectMin    int
		expectMax    int
	}{
		{1, 1, 1},          // 单包应该使用1个协程
		{2, 1, 2},          // 2个包最多2个协程
		{4, 4, 4},          // 等于CPU核心数时
		{20, 4, 20},        // 大量包时应该合理限制
	}
	
	for _, tt := range tests {
		result := GetOptimalWorkerCount(tt.packageCount)
		if result < tt.expectMin || result > tt.expectMax {
			t.Errorf("包数量 %d 的最佳工作协程数 %d 不在预期范围 [%d, %d]", 
				tt.packageCount, result, tt.expectMin, tt.expectMax)
		}
	}
}

// TestCheckParallelCapability 测试并行能力检查
func TestCheckParallelCapability(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	parallelInst := NewParallelInstaller(installer, 4)
	
	// 测试空包列表
	capability := parallelInst.CheckParallelCapability([]string{})
	if capability.Supported {
		t.Error("空包列表不应该支持并行安装")
	}
	
	// 测试单包
	capability = parallelInst.CheckParallelCapability([]string{"test"})
	if capability.Supported {
		t.Error("单包不应该支持并行安装（优势不明显）")
	}
	
	// 注册支持并行的管理器
	mockManager := &MockParallelManager{
		MockPackageManager: NewMockPackageManager("parallel-manager", 1),
	}
	// 重写Name方法返回支持并行的管理器名称
	originalName := mockManager.Name
	mockManager.MockPackageManager.name = "winget"
	installer.RegisterManager(mockManager.MockPackageManager)
	
	// 测试多包（应该支持）
	capability = parallelInst.CheckParallelCapability([]string{"pkg1", "pkg2", "pkg3"})
	if !capability.Supported {
		t.Errorf("多包应该支持并行安装，但检查结果为不支持: %s", capability.Reason)
	}
	
	if capability.RecommendedWorkers <= 0 {
		t.Error("推荐工作协程数应该大于 0")
	}
	
	// 恢复原始名称
	_ = originalName
}

// TestParallelInstaller_SupportsParallel 测试包管理器并行支持检查
func TestParallelInstaller_SupportsParallel(t *testing.T) {
	logger := logrus.New()
	installer := NewInstaller(logger)
	parallelInst := NewParallelInstaller(installer, 4)
	
	tests := []struct {
		managerName string
		expected    bool
	}{
		{"pacman", false},  // Pacman 不支持并行
		{"winget", true},   // Winget 支持并行
		{"yay", false},     // Yay 不支持并行
		{"unknown", false}, // 未知管理器默认不支持
	}
	
	for _, tt := range tests {
		mockManager := NewMockPackageManager(tt.managerName, 1)
		installer.managers = []PackageManager{mockManager} // 重置管理器列表
		
		result := parallelInst.supportsParallel()
		if result != tt.expected {
			t.Errorf("管理器 %s 的并行支持检查结果错误，期望 %v，实际 %v", 
				tt.managerName, tt.expected, result)
		}
	}
}

// TestParallelInstaller_Fallback 测试并行安装回退机制
func TestParallelInstaller_Fallback(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // 静默日志避免测试输出干扰
	installer := NewInstaller(logger)
	
	// 注册不支持并行的管理器
	mockManager := NewMockPackageManager("pacman", 1)
	installer.RegisterManager(mockManager)
	
	parallelInst := NewParallelInstaller(installer, 4)
	
	ctx := context.Background()
	opts := InstallOptions{Quiet: true} // 静默模式避免输出
	packages := []string{"pkg1", "pkg2"}
	
	// 执行并行安装（应该自动回退到串行）
	results, err := parallelInst.InstallPackagesParallel(ctx, packages, opts)
	
	if err != nil {
		t.Errorf("并行安装回退应该成功，但返回错误: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("期望 2 个结果，实际获得 %d 个", len(results))
	}
	
	for _, result := range results {
		if !result.Success {
			t.Errorf("包 %s 安装应该成功", result.PackageName)
		}
	}
}

// TestParallelInstaller_ErrorHandling 测试并行安装错误处理
func TestParallelInstaller_ErrorHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // 静默日志
	installer := NewInstaller(logger)
	
	// 创建会失败的模拟管理器
	mockManager := NewMockPackageManager("winget", 1) // 假装是winget支持并行
	mockManager.SetInstallError(errors.New("模拟安装失败"))
	installer.RegisterManager(mockManager)
	
	parallelInst := NewParallelInstaller(installer, 2)
	
	ctx := context.Background()
	opts := InstallOptions{Quiet: true}
	packages := []string{"pkg1", "pkg2"}
	
	// 执行并行安装（由于不支持并行会回退）
	results, err := parallelInst.InstallPackagesParallel(ctx, packages, opts)
	
	// 即使有错误，也不应该返回错误（错误应该记录在结果中）
	if err != nil {
		t.Errorf("并行安装不应该返回错误，错误应该记录在结果中: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("期望 2 个结果，实际获得 %d 个", len(results))
	}
	
	// 检查结果中的错误
	for _, result := range results {
		if result.Success {
			t.Errorf("包 %s 应该安装失败", result.PackageName)
		}
		if result.Error == nil {
			t.Errorf("失败的包 %s 应该有错误信息", result.PackageName)
		}
	}
}

// BenchmarkParallelVsSerial 并行vs串行性能基准测试
func BenchmarkParallelVsSerial(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // 静默日志
	
	packages := []string{"pkg1", "pkg2", "pkg3", "pkg4", "pkg5", "pkg6"}
	opts := InstallOptions{Quiet: true}
	
	b.Run("Serial", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			installer := NewInstaller(logger)
			mockManager := NewMockParallelManager("test-serial", 1)
			installer.RegisterManager(mockManager.MockPackageManager)
			
			ctx := context.Background()
			_, _ = installer.InstallPackages(ctx, packages, opts)
		}
	})
	
	// 注意：由于当前测试环境中没有真正支持并行的管理器，
	// 这个基准测试主要用于验证测试框架的正确性
	b.Run("Parallel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			installer := NewInstaller(logger)
			mockManager := NewMockParallelManager("test-parallel", 1)
			installer.RegisterManager(mockManager.MockPackageManager)
			
			parallelInst := NewParallelInstaller(installer, 3)
			ctx := context.Background()
			_, _ = parallelInst.InstallPackagesParallel(ctx, packages, opts)
		}
	})
}