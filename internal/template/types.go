// Package template 提供配置文件模板系统，支持跨平台的配置文件生成
package template

import (
	"github.com/bbq191/dotfiles-go/internal/config"
	"github.com/bbq191/dotfiles-go/internal/platform"
)

// TemplateType 定义支持的配置模板类型
type TemplateType string

const (
	TemplateZsh        TemplateType = "zsh"        // ZSH shell 配置模板
	TemplatePowerShell TemplateType = "powershell" // PowerShell 配置模板
)

// GenerateOptions 配置生成过程的控制选项
type GenerateOptions struct {
	OutputDir      string           // 输出目录，为空时使用默认位置
	Templates      []TemplateType   // 要生成的模板类型，为空时自动选择
	DryRun         bool             // 预览模式，不实际创建文件
	Force          bool             // 强制覆盖现有文件
	BackupExisting bool             // 覆盖前创建备份
}

// TemplateContext 模板渲染时的上下文数据
type TemplateContext struct {
	Platform    *platform.PlatformInfo          // 当前运行平台详细信息
	Config      *config.DotfilesConfig          // 完整的 dotfiles 配置数据
	ZshConfig   *config.ZshIntegrationConfig    // Zsh 集成配置快捷引用
	Functions   *config.FunctionsConfig         // 自定义函数配置
	User        config.UserConfig               // 用户基本信息配置
	Paths       config.PathsConfig              // 路径相关配置
	Features    config.FeaturesConfig           // 功能开关配置
	Environment map[string]string               // 环境变量映射表
}

// GenerateResult 单个模板生成操作的结果信息
type GenerateResult struct {
	Template   TemplateType // 生成的模板类型标识
	OutputPath string       // 生成的配置文件完整路径
	Success    bool         // 生成操作是否成功完成
	BackupPath string       // 原有文件的备份路径（如果进行了备份）
	Error      error        // 生成过程中遇到的错误信息
	Generated  bool         // 是否实际生成了文件（预览模式下为 false）
}