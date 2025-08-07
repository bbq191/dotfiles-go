package installer

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestNewYayManager 测试Yay管理器创建
func TestNewYayManager(t *testing.T) {
	logger := logrus.New()
	yayManager := NewYayManager(logger)
	
	if yayManager == nil {
		t.Fatal("NewYayManager 应该返回非空实例")
	}
	
	if yayManager.Name() != "yay" {
		t.Errorf("期望管理器名称为 'yay'，实际为 '%s'", yayManager.Name())
	}
	
	if yayManager.logger != logger {
		t.Error("Yay管理器应该使用提供的logger")
	}
}

// TestYayManager_Priority 测试Yay优先级
func TestYayManager_Priority(t *testing.T) {
	logger := logrus.New()
	yayManager := NewYayManager(logger)
	
	priority := yayManager.Priority()
	if priority != 1 {
		t.Errorf("期望Yay优先级为 1，实际为 %d", priority)
	}
}

// TestYayManager_IsAvailable 测试Yay可用性检查
func TestYayManager_IsAvailable(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // 静默日志
	yayManager := NewYayManager(logger)
	
	// 在测试环境中，我们不能假设yay一定可用
	// 这个测试主要验证方法不会panic
	isAvailable := yayManager.IsAvailable()
	
	// isAvailable 可以是 true 或 false，都是有效的
	_ = isAvailable
}

// TestYayManager_IsInstalled 测试包安装状态检查
func TestYayManager_IsInstalled(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // 静默日志
	yayManager := NewYayManager(logger)
	
	// 测试一个不太可能安装的包
	isInstalled := yayManager.IsInstalled("definitely-not-installed-package-12345")
	
	// 这个包应该不会被安装
	if isInstalled {
		t.Error("不存在的包不应该被检测为已安装")
	}
}

// TestYayManager_Install_DryRun 测试Yay安装功能（仅模拟）
func TestYayManager_Install_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要yay的集成测试")
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // 静默日志
	yayManager := NewYayManager(logger)
	
	// 只有在yay可用时才测试
	if !yayManager.IsAvailable() {
		t.Skip("Yay不可用，跳过安装测试")
	}
	
	ctx := context.Background()
	
	// 测试安装一个已经安装的常见包（避免实际安装）
	// 这通常会快速返回，因为包已经安装
	err := yayManager.Install(ctx, "bash")
	
	if err != nil {
		t.Errorf("安装已存在的包不应该失败: %v", err)
	}
}

// TestYayManager_ParseSearchOutput 测试搜索输出解析
func TestYayManager_ParseSearchOutput(t *testing.T) {
	logger := logrus.New()
	yayManager := NewYayManager(logger)
	
	// 模拟yay搜索输出
	mockOutput := `aur/yay-bin 12.1.0-1 (+1234, 5.67) 
    Yet another Yogurt - An AUR Helper written in Go (precompiled)
core/bash 5.1.016-1
    The GNU Bourne Again shell
extra/git 2.37.1-1
    the fast distributed version control system`
	
	packages := yayManager.parseSearchOutput(mockOutput)
	
	if len(packages) == 0 {
		t.Error("应该解析出至少一个包")
	}
	
	// 检查第一个包的信息
	if len(packages) > 0 {
		pkg := packages[0]
		if pkg.Repository != "aur" {
			t.Errorf("期望仓库为 'aur'，实际为 '%s'", pkg.Repository)
		}
		if pkg.Name != "yay-bin" {
			t.Errorf("期望包名为 'yay-bin'，实际为 '%s'", pkg.Name)
		}
	}
}

// TestYayManager_ParsePackageInfo 测试包信息解析
func TestYayManager_ParsePackageInfo(t *testing.T) {
	logger := logrus.New()
	yayManager := NewYayManager(logger)
	
	// 模拟yay包信息输出
	mockOutput := `Repository      : aur
Name            : yay-bin  
Version         : 12.1.0-1
Description     : Yet another Yogurt - An AUR Helper written in Go
URL             : https://github.com/Jguer/yay
Licenses        : GPL3
Depends On      : pacman  libalpm.so=13  ca-certificates
Make Deps       : None
Installed Size  : 3.34 MiB`
	
	info := yayManager.parsePackageInfo(mockOutput, "yay-bin")
	
	if info.Name != "yay-bin" {
		t.Errorf("期望包名为 'yay-bin'，实际为 '%s'", info.Name)
	}
	
	if info.Repository != "aur" {
		t.Errorf("期望仓库为 'aur'，实际为 '%s'", info.Repository)
	}
	
	if info.Version != "12.1.0-1" {
		t.Errorf("期望版本为 '12.1.0-1'，实际为 '%s'", info.Version)
	}
	
	if len(info.Dependencies) == 0 {
		t.Error("应该解析出依赖信息")
	}
}

// TestAURInstallOptions 测试AUR安装选项结构
func TestAURInstallOptions(t *testing.T) {
	opts := AURInstallOptions{
		NoConfirm:  true,
		SkipReview: false,
	}
	
	if !opts.NoConfirm {
		t.Error("NoConfirm 应该为 true")
	}
	
	if opts.SkipReview {
		t.Error("SkipReview 应该为 false")
	}
}

// BenchmarkYayManager_IsInstalled 基准测试包状态检查性能
func BenchmarkYayManager_IsInstalled(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	yayManager := NewYayManager(logger)
	
	// 只有在yay可用时才运行基准测试
	if !yayManager.IsAvailable() {
		b.Skip("Yay不可用，跳过基准测试")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		yayManager.IsInstalled("bash") // 测试一个通常存在的包
	}
}