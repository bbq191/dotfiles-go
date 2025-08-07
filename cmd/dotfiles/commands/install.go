package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/bbq191/dotfiles-go/internal/config"
	"github.com/bbq191/dotfiles-go/internal/installer"
	"github.com/bbq191/dotfiles-go/internal/interactive"
	"github.com/bbq191/dotfiles-go/internal/platform"
)

var (
	parallel      bool
	maxWorkers    int
	force         bool
	dryRun        bool
	quiet         bool
	interactiveMode bool
)

// installCmd 安装软件包命令
var installCmd = &cobra.Command{
	Use:   "install [packages...]",
	Short: "安装软件包",
	Long: `安装指定的软件包，自动选择最合适的包管理器。

示例:
  dotfiles install                      # 安装所有配置的包
  dotfiles install neovim git fzf     # 安装指定包
  dotfiles install --interactive       # 交互式包选择和安装 ✨
  dotfiles install --force --dry-run  # 预览安装操作
  dotfiles install --parallel          # 并行安装（开发中）`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "交互式包选择和安装")
	installCmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "并行安装 (开发中)")
	installCmd.Flags().IntVarP(&maxWorkers, "max-workers", "w", 0, "最大并行工作数 (0=CPU核心数)")
	installCmd.Flags().BoolVarP(&force, "force", "f", false, "强制重新安装")
	installCmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅显示将要执行的操作")
	installCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "静默模式，不显示进度条")
}

func runInstall(cmd *cobra.Command, args []string) error {
	logger := GetLogger()
	
	// 设置日志级别
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	}
	
	// 检查交互模式
	if interactiveMode {
		return runInteractiveInstall(cmd, args, logger)
	}
	
	logger.Info("🚀 开始软件包安装流程")
	
	// 创建安装器实例
	inst := installer.NewInstaller(logger)
	inst.InitializeManagers()
	
	// 检查是否有可用的包管理器
	availableManagers := inst.GetAvailableManagers()
	if len(availableManagers) == 0 {
		return fmt.Errorf("❌ 未找到可用的包管理器，请确保系统已安装 pacman 或 winget")
	}
	
	// 设置安装选项
	opts := installer.InstallOptions{
		Force:      force,
		DryRun:     dryRun,
		Verbose:    verbose,
		Quiet:      quiet,
		Parallel:   parallel,
		MaxWorkers: maxWorkers,
	}
	
	// 创建上下文（支持取消）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	// 安装包
	if len(args) == 0 {
		return fmt.Errorf("❌ 请指定要安装的包名，例如: dotfiles install neovim git")
	}
	
	logger.Infof("📦 准备安装 %d 个包: %v", len(args), args)
	
	if dryRun {
		fmt.Printf("🔍 预览模式 - 将执行以下操作:\n")
	}
	
	// 检查并行安装能力
	var results []*installer.InstallResult
	var err error
	if opts.Parallel {
		// 创建并行安装器
		parallelInst := installer.NewParallelInstaller(inst, opts.MaxWorkers)
		capability := parallelInst.CheckParallelCapability(args)
		
		if capability.Supported {
			if !opts.Quiet {
				fmt.Printf("⚡ 启用并行安装模式 - %s\n", capability.Reason)
			}
			logger.Infof("使用并行安装: %s", capability.Reason)
			results, err = parallelInst.InstallPackagesParallel(ctx, args, opts)
		} else {
			if !opts.Quiet {
				fmt.Printf("⚠️  并行安装不可用，使用串行模式 - %s\n", capability.Reason)
			}
			logger.Warnf("并行安装不可用: %s，回退到串行模式", capability.Reason)
			results, err = inst.InstallPackages(ctx, args, opts)
		}
	} else {
		// 使用串行安装
		results, err = inst.InstallPackages(ctx, args, opts)
	}
	
	if err != nil {
		logger.Errorf("安装过程中出现错误: %v", err)
		return err
	}
	
	// 检查是否有失败的安装
	failed := 0
	for _, result := range results {
		if !result.Success {
			failed++
		}
	}
	
	if failed > 0 {
		return fmt.Errorf("❌ %d 个包安装失败", failed)
	}
	
	fmt.Println("✅ 所有包安装完成！")
	return nil
}

// runInteractiveInstall 执行交互式安装
func runInteractiveInstall(cmd *cobra.Command, args []string, logger *logrus.Logger) error {
	logger.Info("🎯 启动交互式包选择模式")
	
	// 如果用户在交互模式下还提供了包名参数，提示用户
	if len(args) > 0 {
		logger.Warn("⚠️  交互模式将忽略命令行中指定的包名，请通过界面选择")
	}
	
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	// 检测平台信息
	detector := platform.NewDetector()
	platformInfo, err := detector.DetectPlatform()
	if err != nil {
		return fmt.Errorf("平台检测失败: %w", err)
	}
	
	// 加载配置
	configLoader := config.NewConfigLoader("configs", logger)
	dotfilesConfig, err := configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	
	// 获取包配置（LoadConfig已经加载了）
	packagesConfig := dotfilesConfig.Packages
	if packagesConfig == nil {
		return fmt.Errorf("包配置未正确加载")
	}
	
	// 创建安装器实例
	inst := installer.NewInstaller(logger)
	inst.InitializeManagers()
	
	// 检查是否有可用的包管理器
	availableManagers := inst.GetAvailableManagers()
	if len(availableManagers) == 0 {
		return fmt.Errorf("❌ 未找到可用的包管理器，请确保系统已安装 pacman 或 winget")
	}
	
	logger.Infof("✅ 检测到 %d 个可用包管理器: %v", 
		len(availableManagers), getManagerNames(availableManagers))
	
	// 创建交互式管理器
	interactiveManager := interactive.NewInteractiveManager(
		inst,              // installer
		nil,               // generator (暂时不需要)
		nil,               // xdgManager (暂时不需要)
		dotfilesConfig,    // config
		platformInfo,      // platform
		logger,            // logger
	)
	
	if !interactiveManager.IsEnabled() {
		// 创建一个临时场景来获取详细错误信息
		return fmt.Errorf("❌ 交互式模式在当前环境中不可用\n\n💡 解决方案:\n1. 在真正的终端中运行此命令（如bash、zsh、PowerShell）\n2. 使用非交互式命令: dotfiles install <包名>\n3. 设置环境变量强制启用: DOTFILES_INTERACTIVE=true")
	}
	
	// 创建包选择场景
	packageSelectionScenario := interactive.NewPackageSelectionScenario(
		inst,
		packagesConfig,
		logger,
		interactiveManager.GetTheme(),
	)
	
	// 注册场景
	if err := interactiveManager.RegisterScenario(packageSelectionScenario); err != nil {
		return fmt.Errorf("注册包选择场景失败: %w", err)
	}
	
	// 配置场景选项
	scenarioOptions := map[string]interface{}{
		"force":       force,
		"dry_run":     dryRun,
		"quiet":       quiet,
		"parallel":    parallel,
		"max_workers": maxWorkers,
	}
	
	// 执行交互式包选择场景
	if err := interactiveManager.ExecuteScenario(ctx, "package_selection", scenarioOptions); err != nil {
		return fmt.Errorf("交互式包选择失败: %w", err)
	}
	
	return nil
}

// getManagerNames 获取包管理器名称列表
func getManagerNames(managers []installer.PackageManager) []string {
	var names []string
	for _, manager := range managers {
		names = append(names, manager.Name())
	}
	return names
}