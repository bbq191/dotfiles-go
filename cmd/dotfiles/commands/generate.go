package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bbq191/dotfiles-go/internal/config"
	"github.com/bbq191/dotfiles-go/internal/platform"
	"github.com/bbq191/dotfiles-go/internal/template"
)

var (
	genOutputDir      string
	genTemplates      []string
	genDryRun         bool
	genForce          bool
	genBackupExisting bool
)

// generateCmd 生成配置文件命令
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "生成配置文件",
	Long: `基于模板和配置生成各种 shell 配置文件。

支持的配置类型:
  • PowerShell Profile
  • Zsh 配置文件
  • 环境变量设置
  • XDG 目录配置

示例:
  dotfiles generate                   # 生成所有配置
  dotfiles generate --templates=zsh   # 只生成 Zsh 配置
  dotfiles generate --output-dir=/tmp # 指定输出目录`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&genOutputDir, "output-dir", "o", "", "输出目录")
	generateCmd.Flags().StringSliceVarP(&genTemplates, "templates", "t", []string{}, "指定模板类型 (zsh,powershell,xdg)")
	generateCmd.Flags().BoolVar(&genDryRun, "dry-run", false, "预览模式，不实际生成文件")
	generateCmd.Flags().BoolVar(&genForce, "force", false, "强制覆盖现有文件")
	generateCmd.Flags().BoolVar(&genBackupExisting, "backup", false, "备份现有文件")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	logger := GetLogger()
	
	logger.Info("🎯 开始配置文件生成流程")
	
	// 加载配置
	configLoader := config.NewConfigLoader("configs", logger)
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		logger.Errorf("加载配置失败: %v", err)
		return fmt.Errorf("加载配置失败: %w", err)
	}
	
	// 检测平台信息
	detector := platform.NewDetector()
	platformInfo, err := detector.DetectPlatform()
	if err != nil {
		logger.Warnf("平台检测失败，使用基本信息: %v", err)
		// 使用基本平台信息
		platformInfo = &platform.PlatformInfo{
			OS:           "unknown",
			Architecture: "unknown",
		}
	}
	
	// 确定模板目录路径
	templateDir := "templates"
	if filepath.IsAbs(templateDir) == false {
		// 相对路径，相对于可执行文件
		templateDir = filepath.Join(".", templateDir)
	}
	
	// 创建生成器
	generator := template.NewGenerator(templateDir, cfg, platformInfo, logger)
	
	// 验证模板文件
	if err := generator.ValidateTemplates(); err != nil {
		logger.Warnf("模板验证失败: %v", err)
		logger.Info("部分模板可能不可用，将跳过")
	}
	
	// 转换模板类型
	var templateTypes []template.TemplateType
	if len(genTemplates) > 0 {
		for _, t := range genTemplates {
			templateTypes = append(templateTypes, template.TemplateType(strings.ToLower(t)))
		}
		logger.Infof("指定模板: %v", templateTypes)
	} else {
		logger.Info("将生成推荐的模板")
	}
	
	// 设置生成选项
	options := template.GenerateOptions{
		OutputDir:       genOutputDir,
		Templates:       templateTypes,
		DryRun:          genDryRun,
		Force:           genForce,
		BackupExisting:  genBackupExisting,
	}
	
	if genDryRun {
		logger.Info("📋 预览模式：将显示生成内容而不创建文件")
	}
	
	// 执行生成
	results, err := generator.GenerateConfigs(options)
	if err != nil {
		logger.Errorf("生成配置失败: %v", err)
		return fmt.Errorf("生成配置失败: %w", err)
	}
	
	// 显示结果统计
	var successCount, failureCount int
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
			logger.Errorf("❌ %s 生成失败: %v", result.Template, result.Error)
		}
	}
	
	// 输出总结
	if genDryRun {
		fmt.Printf("\n📋 预览完成！共检查 %d 个模板\n", len(results))
	} else {
		fmt.Printf("\n✨ 生成完成！成功: %d, 失败: %d\n", successCount, failureCount)
	}
	
	if failureCount > 0 {
		return fmt.Errorf("部分配置生成失败，请检查错误信息")
	}
	
	return nil
}