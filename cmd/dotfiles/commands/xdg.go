package commands

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bbq191/dotfiles-go/internal/xdg"
)

var (
	migrate bool
)

// xdgCmd XDG 配置迁移命令
var xdgCmd = &cobra.Command{
	Use:   "xdg",
	Short: "XDG 配置迁移",
	Long: `迁移现有配置文件到 XDG Base Directory 规范目录。

XDG 规范定义了应用程序配置、数据、缓存等文件的标准存储位置，
有助于保持家目录的整洁和组织。

注意:
  • 要生成 XDG 配置，请使用: dotfiles generate --templates=xdg
  • 此命令仅用于迁移现有的配置文件到 XDG 目录

示例:
  dotfiles generate --templates=xdg  # 生成 XDG 配置脚本
  dotfiles xdg migrate               # 迁移现有配置到 XDG 目录`,
}


// xdgMigrateCmd XDG 迁移子命令
var xdgMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "迁移到 XDG 目录",
	Long: `将现有的配置文件迁移到 XDG 规范目录。

支持迁移的应用:
  • Zsh 配置文件
  • Git 配置
  • Vim/Neovim 配置
  • 其他支持 XDG 的应用`,
	RunE: runXDGMigrate,
}

func init() {
	rootCmd.AddCommand(xdgCmd)
	xdgCmd.AddCommand(xdgMigrateCmd)

	xdgMigrateCmd.Flags().BoolVarP(&migrate, "force", "f", false, "强制迁移（覆盖现有文件）")
}


func runXDGMigrate(cmd *cobra.Command, args []string) error {
	logger := GetLogger()
	
	logger.Info("🚀 开始 XDG 配置迁移")
	
	// 创建XDG管理器
	xdgManager := xdg.NewManager(logger, runtime.GOOS)
	
	// 确保XDG目录存在
	if err := xdgManager.EnsureDirectories(); err != nil {
		return fmt.Errorf("创建 XDG 目录失败: %w", err)
	}
	
	// 首先进行合规性检查
	logger.Info("🔍 检查当前配置的 XDG 合规性...")
	issues, err := xdgManager.CheckCompliance()
	if err != nil {
		return fmt.Errorf("合规性检查失败: %w", err)
	}
	
	if len(issues) == 0 {
		fmt.Println("✅ 当前配置已完全符合 XDG 规范")
		return nil
	}
	
	fmt.Printf("📋 发现 %d 个需要迁移的项目:\n", len(issues))
	for i, issue := range issues {
		fmt.Printf("[%d] %s: %s\n", i+1, issue.Application, issue.Description)
		if issue.CurrentPath != "" {
			fmt.Printf("    当前路径: %s\n", issue.CurrentPath)
		}
		if issue.RecommendedPath != "" {
			fmt.Printf("    推荐路径: %s\n", issue.RecommendedPath)
		}
	}
	
	// 确定要迁移的应用列表
	var applications []string
	if len(args) > 0 {
		applications = args
	} else {
		// 从合规性问题中提取应用名称
		appSet := make(map[string]bool)
		for _, issue := range issues {
			if issue.Application != "system" && issue.AutoFixable {
				appSet[issue.Application] = true
			}
		}
		for app := range appSet {
			applications = append(applications, app)
		}
	}
	
	if len(applications) == 0 {
		fmt.Println("📝 没有可自动迁移的应用，请手动设置环境变量")
		return nil
	}
	
	// 计划迁移任务
	logger.Infof("📋 规划迁移任务，应用: %v", applications)
	tasks, err := xdgManager.PlanMigration(applications)
	if err != nil {
		return fmt.Errorf("规划迁移失败: %w", err)
	}
	
	if len(tasks) == 0 {
		fmt.Println("📝 没有找到需要迁移的配置文件")
		fmt.Println("💡 要生成 XDG 配置脚本，请使用: dotfiles generate --templates=xdg")
		return nil
	}
	
	// 设置迁移选项
	options := xdg.MigrationOptions{
		Force:         migrate,
		Backup:        !migrate, // 非强制模式时创建备份
		DryRun:        false,
		Interactive:   false,
		Parallel:      false,    // 串行执行更安全
		IgnoreErrors:  false,
		Verbose:       true,
	}
	
	// 预演迁移
	fmt.Printf("\n📋 迁移预演 (%d 个任务):\n", len(tasks))
	previewOptions := options
	previewOptions.DryRun = true
	if err := xdgManager.ExecuteMigration(tasks, previewOptions); err != nil {
		return fmt.Errorf("迁移预演失败: %w", err)
	}
	
	// 询问用户确认（在实际场景中可以使用交互式确认）
	if !migrate {
		fmt.Println("\n⚠️  即将执行上述迁移操作")
		fmt.Println("💡 使用 --force 标志跳过确认并强制执行")
		fmt.Println("💡 将自动创建备份到 ~/.local/share/dotfiles/xdg-backup/")
	}
	
	// 执行迁移
	logger.Info("⚡ 开始执行迁移...")
	if err := xdgManager.ExecuteMigration(tasks, options); err != nil {
		return fmt.Errorf("执行迁移失败: %w", err)
	}
	
	// 显示迁移后建议
	fmt.Printf("\n🎉 XDG 迁移完成！\n")
	fmt.Println("💡 现在可以生成 XDG 配置脚本: dotfiles generate --templates=xdg")
	fmt.Println("💡 或者手动在 shell 配置文件中设置以下环境变量:")
	
	directories := []xdg.XDGDirectory{
		xdg.ConfigHome, xdg.DataHome, xdg.StateHome, xdg.CacheHome,
	}
	
	for _, dirType := range directories {
		path, err := xdgManager.GetXDGPath(dirType)
		if err != nil {
			continue
		}
		envVarName := fmt.Sprintf("XDG_%s_HOME", strings.ToUpper(dirType.String()))
		fmt.Printf("export %s=%s\n", envVarName, path)
	}
	
	fmt.Println("\n🔄 重启 shell 或执行 'source ~/.zshrc' 以应用更改")
	
	logger.Info("✅ XDG 迁移完成")
	return nil
}