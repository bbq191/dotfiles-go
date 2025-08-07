package interactive

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/bbq191/dotfiles-go/internal/config"
	"github.com/bbq191/dotfiles-go/internal/installer"
	"github.com/bbq191/dotfiles-go/internal/platform"
	"github.com/bbq191/dotfiles-go/internal/template"
	"github.com/bbq191/dotfiles-go/internal/xdg"
)

// InteractiveScenario 交互场景通用接口
type InteractiveScenario interface {
	// 场景基本信息
	Name() string                             // 场景名称
	Description() string                      // 场景描述
	Prerequisites() []string                  // 前置条件检查
	
	// 场景执行流程
	CanExecute(ctx context.Context) (bool, error)  // 是否可执行
	Execute(ctx context.Context) error             // 执行场景
	Preview() (string, error)                     // 预览执行效果
	
	// 配置和状态
	Configure(options map[string]interface{}) error  // 配置场景参数
	GetStatus() ScenarioStatus                      // 获取当前状态
}

// ScenarioStatus 场景执行状态
type ScenarioStatus int

const (
	StatusNotReady ScenarioStatus = iota  // 未就绪
	StatusReady                           // 就绪
	StatusRunning                         // 执行中  
	StatusCompleted                       // 已完成
	StatusFailed                          // 执行失败
	StatusCancelled                       // 已取消
)

func (s ScenarioStatus) String() string {
	switch s {
	case StatusNotReady:
		return "not_ready"
	case StatusReady:
		return "ready"
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// InteractiveManager 交互式管理器
type InteractiveManager struct {
	// 依赖注入 - 复用现有系统
	installer    *installer.Installer
	generator    *template.Generator
	xdgManager   *xdg.Manager
	config       *config.DotfilesConfig
	platform     *platform.PlatformInfo
	logger       *logrus.Logger
	
	// 交互式功能
	theme        *UITheme
	scenarios    map[string]InteractiveScenario
	enabled      bool
	
	// 运行时状态
	currentScenario InteractiveScenario
	startTime       time.Time
}

// UITheme UI主题配置
type UITheme struct {
	// 颜色配置
	PrimaryColor    string `json:"primary_color"`     // 主色调
	SecondaryColor  string `json:"secondary_color"`   // 辅色调
	AccentColor     string `json:"accent_color"`      // 强调色
	ErrorColor      string `json:"error_color"`       // 错误色
	SuccessColor    string `json:"success_color"`     // 成功色
	WarningColor    string `json:"warning_color"`     // 警告色
	
	// 图标配置
	Icons           IconSet `json:"icons"`
	
	// 布局配置
	MaxWidth        int     `json:"max_width"`         // 最大宽度
	Padding         int     `json:"padding"`           // 内边距
	EnableEmojis    bool    `json:"enable_emojis"`     // 是否启用emoji
	
	// 交互配置
	ShowProgress    bool    `json:"show_progress"`     // 显示进度
	ShowPreview     bool    `json:"show_preview"`      // 显示预览
	ConfirmActions  bool    `json:"confirm_actions"`   // 确认操作
}

// IconSet 图标集合
type IconSet struct {
	Success      string `json:"success"`       // ✅
	Error        string `json:"error"`         // ❌
	Warning      string `json:"warning"`       // ⚠️
	Info         string `json:"info"`          // ℹ️
	Question     string `json:"question"`      // ❓
	Package      string `json:"package"`       // 📦
	Category     string `json:"category"`      // 📁
	Search       string `json:"search"`        // 🔍
	Install      string `json:"install"`       // ⬇️
	Configure    string `json:"configure"`     // ⚙️
	Preview      string `json:"preview"`       // 👁️
	Migration    string `json:"migration"`     // 🔄
}

// NewInteractiveManager 创建交互式管理器
func NewInteractiveManager(
	installer *installer.Installer,
	generator *template.Generator,
	xdgManager *xdg.Manager,
	config *config.DotfilesConfig,
	platform *platform.PlatformInfo,
	logger *logrus.Logger,
) *InteractiveManager {
	
	// 检查是否启用交互功能
	enabled := isInteractiveEnabled()
	
	return &InteractiveManager{
		installer:   installer,
		generator:   generator,
		xdgManager:  xdgManager,
		config:      config,
		platform:    platform,
		logger:      logger,
		theme:       getDefaultTheme(),
		scenarios:   make(map[string]InteractiveScenario),
		enabled:     enabled,
	}
}

// RegisterScenario 注册交互场景
func (m *InteractiveManager) RegisterScenario(scenario InteractiveScenario) error {
	if scenario == nil {
		return fmt.Errorf("scenario cannot be nil")
	}
	
	name := scenario.Name()
	if name == "" {
		return fmt.Errorf("scenario name cannot be empty")
	}
	
	if _, exists := m.scenarios[name]; exists {
		return fmt.Errorf("scenario %s already registered", name)
	}
	
	m.scenarios[name] = scenario
	m.logger.Debugf("已注册交互场景: %s", name)
	return nil
}

// GetScenario 获取交互场景
func (m *InteractiveManager) GetScenario(name string) (InteractiveScenario, error) {
	scenario, exists := m.scenarios[name]
	if !exists {
		return nil, fmt.Errorf("scenario %s not found", name)
	}
	return scenario, nil
}

// ListScenarios 列出所有可用场景
func (m *InteractiveManager) ListScenarios() []string {
	var names []string
	for name := range m.scenarios {
		names = append(names, name)
	}
	return names
}

// ExecuteScenario 执行指定场景
func (m *InteractiveManager) ExecuteScenario(ctx context.Context, scenarioName string, options map[string]interface{}) error {
	if !m.enabled {
		reason := getInteractiveDisabledReason()
		return fmt.Errorf("交互模式不可用: %s\n\n💡 解决方案:\n1. 在真正的终端中运行此命令（如bash、zsh、PowerShell）\n2. 使用非交互式命令: dotfiles install <包名>\n3. 设置环境变量强制启用: DOTFILES_INTERACTIVE=true", reason)
	}
	
	scenario, err := m.GetScenario(scenarioName)
	if err != nil {
		return err
	}
	
	// 检查前置条件
	if canExecute, err := scenario.CanExecute(ctx); err != nil {
		return fmt.Errorf("failed to check prerequisites: %w", err)
	} else if !canExecute {
		return fmt.Errorf("scenario prerequisites not met")
	}
	
	// 配置场景
	if options != nil {
		if err := scenario.Configure(options); err != nil {
			return fmt.Errorf("failed to configure scenario: %w", err)
		}
	}
	
	// 执行场景
	m.currentScenario = scenario
	m.startTime = time.Now()
	
	m.logger.Infof("🚀 开始执行交互场景: %s", scenario.Name())
	
	if err := scenario.Execute(ctx); err != nil {
		m.logger.Errorf("场景执行失败: %v", err)
		return err
	}
	
	duration := time.Since(m.startTime)
	m.logger.Infof("✅ 场景执行完成，耗时: %v", duration)
	
	return nil
}

// IsEnabled 检查交互功能是否启用
func (m *InteractiveManager) IsEnabled() bool {
	return m.enabled
}

// GetTheme 获取当前主题
func (m *InteractiveManager) GetTheme() *UITheme {
	return m.theme
}

// SetTheme 设置主题
func (m *InteractiveManager) SetTheme(theme *UITheme) {
	if theme != nil {
		m.theme = theme
	}
}


// isInteractiveEnabled 检查是否启用交互功能
func isInteractiveEnabled() bool {
	// 检查环境变量
	if disabled := os.Getenv("DOTFILES_INTERACTIVE"); disabled != "" {
		return strings.ToLower(disabled) != "false" && disabled != "0"
	}
	
	// 检查是否为完整的TTY环境
	if !isatty() {
		return false
	}
	
	// 默认启用
	return true
}

// getInteractiveDisabledReason 获取交互功能禁用原因
func getInteractiveDisabledReason() string {
	// 检查环境变量
	if disabled := os.Getenv("DOTFILES_INTERACTIVE"); disabled != "" {
		if strings.ToLower(disabled) == "false" || disabled == "0" {
			return "环境变量 DOTFILES_INTERACTIVE 被设置为禁用"
		}
	}
	
	// 详细检查TTY状态
	if fi, err := os.Stdin.Stat(); err != nil || (fi.Mode() & os.ModeCharDevice) == 0 {
		return "标准输入不是终端设备，请在真正的终端中运行此命令"
	}
	
	if fi, err := os.Stdout.Stat(); err != nil || (fi.Mode() & os.ModeCharDevice) == 0 {
		return "标准输出不是终端设备，当前环境不支持交互模式"
	}
	
	if fi, err := os.Stderr.Stat(); err != nil || (fi.Mode() & os.ModeCharDevice) == 0 {
		return "标准错误不是终端设备，当前环境不支持交互模式"
	}
	
	return "未知原因"
}

// isatty 检查是否为完整的终端环境
func isatty() bool {
	// 检查标准输入是否为终端
	if fi, err := os.Stdin.Stat(); err != nil || (fi.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	
	// 检查标准输出是否为终端（survey库需要）
	if fi, err := os.Stdout.Stat(); err != nil || (fi.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	
	// 检查标准错误是否为终端
	if fi, err := os.Stderr.Stat(); err != nil || (fi.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	
	return true
}

// getDefaultTheme 获取默认主题
func getDefaultTheme() *UITheme {
	return &UITheme{
		PrimaryColor:    "#0366d6",   // GitHub蓝
		SecondaryColor:  "#586069",   // 灰色
		AccentColor:     "#28a745",   // 绿色
		ErrorColor:      "#d73a49",   // 红色
		SuccessColor:    "#28a745",   // 绿色
		WarningColor:    "#ffc107",   // 黄色
		
		Icons: IconSet{
			Success:      "✅",
			Error:        "❌",
			Warning:      "⚠️",
			Info:         "ℹ️",
			Question:     "❓",
			Package:      "📦",
			Category:     "📁",
			Search:       "🔍",
			Install:      "⬇️",
			Configure:    "⚙️",
			Preview:      "👁️",
			Migration:    "🔄",
		},
		
		MaxWidth:        120,
		Padding:         2,
		EnableEmojis:    true,
		ShowProgress:    true,
		ShowPreview:     true,
		ConfirmActions:  true,
	}
}

// ScenarioContext 场景上下文
type ScenarioContext struct {
	Platform     *platform.PlatformInfo  `json:"platform"`
	Config       *config.DotfilesConfig  `json:"config"`
	Theme        *UITheme                `json:"theme"`
	Options      map[string]interface{}  `json:"options"`
	Logger       *logrus.Logger          `json:"-"`
}

// InteractionResult 交互结果
type InteractionResult struct {
	Success      bool                   `json:"success"`
	Data         map[string]interface{} `json:"data"`
	Message      string                 `json:"message"`
	Duration     time.Duration          `json:"duration"`
	Error        error                  `json:"error,omitempty"`
}