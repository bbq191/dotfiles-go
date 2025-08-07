package commands

import (
	"fmt"

	"github.com/bbq191/dotfiles-go/internal/platform"
	"github.com/spf13/cobra"
)

// infoCmd 显示系统信息命令
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "显示系统和平台信息",
	Long: `显示当前系统的详细信息，包括：

• 操作系统和架构
• WSL 环境检测结果
• PowerShell 版本信息
• Linux 发行版信息
• 可用的包管理器
• 推荐的配置路径

该命令主要用于诊断和了解当前运行环境。`,
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	logger := GetLogger()
	logger.Info("正在检测平台信息...")
	
	detector := platform.NewDetector()
	info, err := detector.DetectPlatform()
	if err != nil {
		return fmt.Errorf("平台检测失败: %w", err)
	}
	
	// 显示平台基础信息
	fmt.Println("=== 平台信息 ===")
	fmt.Println(info.String())
	fmt.Println()
	
	// 显示包管理器信息
	fmt.Println("=== 包管理器 ===")
	recommended := info.GetRecommendedPackageManagers()
	if len(recommended) > 0 {
		fmt.Printf("推荐的包管理器: %v\n", recommended)
	}
	
	available := platform.GetAvailablePackageManagers()
	if len(available) > 0 {
		fmt.Printf("可用的包管理器: %v\n", available)
	} else {
		fmt.Println("未检测到任何包管理器")
	}
	fmt.Println()
	
	// 显示配置路径建议
	fmt.Println("=== 配置路径建议 ===")
	paths := info.GetConfigPaths()
	for name, path := range paths {
		fmt.Printf("%s: %s\n", name, path)
	}
	fmt.Println()
	
	// 显示功能支持情况
	fmt.Println("=== 功能支持 ===")
	fmt.Printf("WSL 环境: %v\n", info.IsWSLEnvironment())
	fmt.Printf("WSL2 环境: %v\n", info.IsWSL2Environment())
	fmt.Printf("PowerShell 支持: %v\n", info.SupportsPowerShell())
	
	// 测试常见包管理器支持
	managers := []string{"pacman", "yay", "apt", "winget", "scoop"}
	fmt.Println("包管理器支持:")
	for _, manager := range managers {
		supported := info.SupportsPackageManager(manager)
		fmt.Printf("  %s: %v\n", manager, supported)
	}
	
	logger.Info("平台信息检测完成")
	return nil
}