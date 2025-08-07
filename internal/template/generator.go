package template

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/bbq191/dotfiles-go/internal/config"
	"github.com/bbq191/dotfiles-go/internal/platform"
)

// Generator 高级配置文件生成器
type Generator struct {
	engine       *Engine                   // 底层模板引擎
	config       *config.DotfilesConfig    // 完整的 dotfiles 配置
	platformInfo *platform.PlatformInfo   // 当前运行平台信息
	logger       *logrus.Logger            // 日志记录器
}

// NewGenerator 创建新的配置生成器实例
func NewGenerator(templateDir string, cfg *config.DotfilesConfig, info *platform.PlatformInfo, logger *logrus.Logger) *Generator {
	engine := NewEngine(templateDir, logger) // 创建底层模板引擎
	
	return &Generator{
		engine:       engine, // 设置模板引擎
		config:       cfg,    // 设置配置数据
		platformInfo: info,   // 设置平台信息
		logger:       logger, // 设置日志记录器
	}
}

// GenerateConfigs 执行批量配置文件生成操作
func (g *Generator) GenerateConfigs(options GenerateOptions) ([]GenerateResult, error) {
	var results []GenerateResult
	
	templateTypes := options.Templates // 获取指定的模板类型
	if len(templateTypes) == 0 {        // 未指定时使用推荐模板
		templateTypes = g.getRecommendedTemplates()
	}
	
	g.logger.Infof("准备生成 %d 个配置文件", len(templateTypes))
	
	// 预加载所有需要的模板文件
	if err := g.engine.LoadTemplates(templateTypes...); err != nil {
		return nil, fmt.Errorf("加载模板失败: %w", err)
	}
	
	context := g.createTemplateContext() // 创建模板渲染上下文
	
	// 逐一生成配置文件
	for _, templateType := range templateTypes {
		result := g.generateSingleConfig(templateType, context, options) // 生成单个配置文件
		results = append(results, result)                                // 收集结果
		
		// 记录生成状态
		if result.Success {
			g.logger.Infof("✅ %s 配置生成成功: %s", templateType, result.OutputPath)
		} else {
			g.logger.Errorf("❌ %s 配置生成失败: %v", templateType, result.Error)
		}
	}
	
	return results, nil
}

// generateSingleConfig 生成单个配置文件
func (g *Generator) generateSingleConfig(templateType TemplateType, context *TemplateContext, options GenerateOptions) GenerateResult {
	result := GenerateResult{
		Template:  templateType,
		Generated: !options.DryRun,
	}
	
	// 确定输出路径
	outputPath, err := g.getOutputPath(templateType, options.OutputDir)
	if err != nil {
		result.Error = err
		return result
	}
	result.OutputPath = outputPath
	
	// 检查文件是否已存在
	if !options.Force && !options.DryRun {
		if _, err := os.Stat(outputPath); err == nil {
			if options.BackupExisting {
				backupPath := outputPath + ".backup"
				if err := os.Rename(outputPath, backupPath); err != nil {
					result.Error = fmt.Errorf("备份现有文件失败: %w", err)
					return result
				}
				result.BackupPath = backupPath
				g.logger.Infof("已备份现有文件: %s", backupPath)
			} else {
				result.Error = fmt.Errorf("文件已存在，使用 --force 强制覆盖: %s", outputPath)
				return result
			}
		}
	}
	
	// 预览模式
	if options.DryRun {
		g.logger.Infof("📋 [预览] 将生成 %s: %s", templateType, outputPath)
		result.Success = true
		return result
	}
	
	// 生成配置文件
	if err := g.engine.Generate(templateType, context, outputPath); err != nil {
		result.Error = fmt.Errorf("生成配置失败: %w", err)
		return result
	}
	
	result.Success = true
	return result
}

// createTemplateContext 创建模板上下文
func (g *Generator) createTemplateContext() *TemplateContext {
	return &TemplateContext{
		Platform:    g.platformInfo,
		Config:      g.config,
		ZshConfig:   g.config.ZshConfig,
		Functions:   g.config.Functions,
		User:        g.config.User,
		Paths:       g.config.Paths,
		Features:    g.config.Features,
		Environment: g.config.Environment,
	}
}

// getRecommendedTemplates 获取推荐的模板类型
func (g *Generator) getRecommendedTemplates() []TemplateType {
	var templates []TemplateType
	
	// 基于平台信息推荐模板
	if g.platformInfo.OS == "linux" || (g.platformInfo.WSL != nil && g.platformInfo.WSL.IsWSL) {
		templates = append(templates, TemplateZsh)
	}
	
	if g.platformInfo.OS == "windows" || (g.platformInfo.WSL != nil && g.platformInfo.WSL.IsWSL) {
		if g.platformInfo.PowerShell != nil {
			templates = append(templates, TemplatePowerShell)
		}
	}
	
	// 如果没有检测到特定平台，默认生成所有
	if len(templates) == 0 {
		templates = []TemplateType{TemplateZsh, TemplatePowerShell}
	}
	
	g.logger.Debugf("推荐模板: %v", templates)
	return templates
}

// getOutputPath 获取输出路径
func (g *Generator) getOutputPath(templateType TemplateType, customOutputDir string) (string, error) {
	var filename string
	var defaultDir string
	
	switch templateType {
	case TemplateZsh:
		filename = ".zshrc"
		if g.config.ZshConfig != nil && g.config.ZshConfig.XDGDirectories.Enabled {
			// 使用 XDG 标准路径
			configHome := g.config.ZshConfig.XDGDirectories.ConfigHome.Get("linux")
			if configHome == "" {
				configHome = "$HOME/.config"
			}
			defaultDir = filepath.Join(os.ExpandEnv(configHome), "zsh")
		} else {
			defaultDir = "$HOME"
		}
		
	case TemplatePowerShell:
		filename = "Microsoft.PowerShell_profile.ps1"
		if g.platformInfo.WSL != nil && g.platformInfo.WSL.IsWSL && g.platformInfo.WSL.WindowsHome != "" {
			// WSL环境中，生成到Windows用户目录
			defaultDir = filepath.Join(g.platformInfo.WSL.WindowsHome, "Documents", "PowerShell")
		} else if runtime.GOOS == "windows" {
			defaultDir = "$HOME\\Documents\\PowerShell"
		} else {
			// Linux环境下的备选路径
			defaultDir = "$HOME/.config/powershell"
		}
		
	default:
		return "", fmt.Errorf("未知的模板类型: %s", templateType)
	}
	
	// 使用自定义输出目录或默认目录
	outputDir := customOutputDir
	if outputDir == "" {
		outputDir = os.ExpandEnv(defaultDir)
	}
	
	return filepath.Join(outputDir, filename), nil
}

// ValidateTemplates 验证模板文件
func (g *Generator) ValidateTemplates(templateTypes ...TemplateType) error {
	if len(templateTypes) == 0 {
		templateTypes = []TemplateType{TemplateZsh, TemplatePowerShell}
	}
	
	for _, templateType := range templateTypes {
		templatePath := g.engine.getTemplatePath(templateType)
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			g.logger.Warnf("模板文件不存在: %s", templatePath)
			continue
		}
		
		// 尝试加载模板进行语法验证
		if err := g.engine.loadTemplate(templateType); err != nil {
			return fmt.Errorf("模板 %s 语法错误: %w", templateType, err)
		}
		
		g.logger.Debugf("模板验证通过: %s", templateType)
	}
	
	return nil
}

// GetTemplatePreview 获取模板预览
func (g *Generator) GetTemplatePreview(templateType TemplateType, maxLines int) (string, error) {
	templatePath := g.engine.getTemplatePath(templateType)
	
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("读取模板文件失败: %w", err)
	}
	
	lines := strings.Split(string(content), "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "...")
	}
	
	return strings.Join(lines, "\n"), nil
}