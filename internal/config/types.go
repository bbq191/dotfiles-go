package config

import (
	"encoding/json"
	"fmt"
)

// DotfilesConfig 主配置结构
type DotfilesConfig struct {
	Version     string                `json:"version,omitempty" validate:"omitempty,semver"`
	User        UserConfig            `json:"user" validate:"required"`
	Paths       PathsConfig           `json:"paths"`
	Environment map[string]string     `json:"environment"`
	Features    FeaturesConfig        `json:"features"`
	ZshConfig   *ZshIntegrationConfig `json:"-"` // 从单独文件加载
	Packages    *PackagesConfig       `json:"-"` // 从单独文件加载
	Functions   *FunctionsConfig      `json:"-"` // 从单独文件加载
}

// UserConfig 用户配置
type UserConfig struct {
	Name    string `json:"name" validate:"required,min=1"`
	Email   string `json:"email" validate:"required,email"`
	Editor  string `json:"editor,omitempty"`
	Browser string `json:"browser,omitempty"`
}

// PathsConfig 路径配置
type PathsConfig struct {
	Projects  PathValue `json:"projects"`
	Dotfiles  PathValue `json:"dotfiles"`
	Scripts   PathValue `json:"scripts,omitempty"`
	Templates PathValue `json:"templates,omitempty"`
}

// PathValue 路径值 - 支持字符串或平台特定对象
type PathValue struct {
	Default  string            `json:"-"`
	Platform map[string]string `json:"-"`
}

// UnmarshalJSON 自定义 JSON 解析
func (p *PathValue) UnmarshalJSON(data []byte) error {
	// 尝试解析为字符串
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		p.Default = str
		p.Platform = nil
		return nil
	}

	// 尝试解析为对象
	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err == nil {
		p.Platform = obj
		p.Default = ""
		return nil
	}

	return fmt.Errorf("invalid path value format")
}

// MarshalJSON 自定义 JSON 序列化
func (p PathValue) MarshalJSON() ([]byte, error) {
	if p.Platform != nil {
		return json.Marshal(p.Platform)
	}
	return json.Marshal(p.Default)
}

// Get 获取指定平台的路径值
func (p PathValue) Get(platform string) string {
	if p.Platform != nil {
		if val, ok := p.Platform[platform]; ok {
			return val
		}
		// 尝试通用键
		if val, ok := p.Platform["default"]; ok {
			return val
		}
	}
	return p.Default
}

// FeaturesConfig 功能配置
type FeaturesConfig struct {
	GitIntegration    bool `json:"git_integration"`
	NodejsManagement  bool `json:"nodejs_management"`
	PythonManagement  bool `json:"python_management"`
	CompletionCache   bool `json:"completion_cache,omitempty"`
	AsyncLoading      bool `json:"async_loading,omitempty"`
	PathDeduplication bool `json:"path_deduplication,omitempty"`
}

// ZshIntegrationConfig Zsh 集成配置（从 zsh_integration.json 加载）
type ZshIntegrationConfig struct {
	Proxy                   ProxyConfig                     `json:"proxy"`
	XDGDirectories          XDGConfig                       `json:"xdg_directories"`
	HistoryAdvanced         HistoryConfig                   `json:"history_advanced"`
	CompletionAdvanced      CompletionConfig                `json:"completion_advanced"`
	ModernTools             ModernToolsConfig               `json:"modern_tools"`
	DevelopmentEnvironments map[string]map[string]PathValue `json:"development_environments"`
	FzfConfig               FzfConfig                       `json:"fzf_config"`
	Keybindings             KeybindingsConfig               `json:"keybindings"`
	VersionManagers         map[string]VersionManager       `json:"version_managers"`
	GitTools                map[string]GitTool              `json:"git_tools"`
	ExternalTools           ExternalToolsConfig             `json:"external_tools"`
	Performance             PerformanceConfig               `json:"performance"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Enabled       bool                    `json:"enabled"`
	AutoDetect    bool                    `json:"auto_detect"`
	Profiles      map[string]ProxyProfile `json:"profiles"`
	ActiveProfile string                  `json:"active_profile"`
}

// ProxyProfile 代理配置文件
type ProxyProfile struct {
	HTTPSProxy string `json:"https_proxy"`
	HTTPProxy  string `json:"http_proxy"`
	AllProxy   string `json:"all_proxy"`
	NoProxy    string `json:"no_proxy"`
}

// XDGConfig XDG 目录配置
type XDGConfig struct {
	Enabled    bool      `json:"enabled"`
	ConfigHome PathValue `json:"config_home"`
	DataHome   PathValue `json:"data_home"`
	StateHome  PathValue `json:"state_home"`
	CacheHome  PathValue `json:"cache_home"`
	RuntimeDir PathValue `json:"runtime_dir"`
	UserBin    PathValue `json:"user_bin"`
}

// HistoryConfig 历史记录配置
type HistoryConfig struct {
	File      string                 `json:"file"`
	BackupDir string                 `json:"backup_dir"`
	Size      int                    `json:"size"`
	SaveSize  int                    `json:"save_size"`
	Options   map[string]interface{} `json:"options"`
}

// CompletionConfig 自动完成配置
type CompletionConfig struct {
	CachePath string                 `json:"cache_path"`
	DumpFile  string                 `json:"dump_file"`
	Options   map[string]interface{} `json:"options"`
	Styles    map[string]interface{} `json:"styles"`
}

// ModernToolsConfig 现代工具替代配置
type ModernToolsConfig struct {
	Replacements map[string]ToolReplacement `json:"replacements"`
}

// ToolReplacement 工具替代配置
type ToolReplacement struct {
	Tool        string            `json:"tool"`
	Fallback    string            `json:"fallback,omitempty"`
	Aliases     map[string]string `json:"aliases,omitempty"`
	InitCommand string            `json:"init_command,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
}

// FzfConfig FZF 配置
type FzfConfig struct {
	Enabled  bool              `json:"enabled"`
	Commands map[string]string `json:"commands"`
	Theme    interface{}       `json:"theme"`
	Preview  map[string]string `json:"preview"`
}

// KeybindingsConfig 键绑定配置
type KeybindingsConfig struct {
	HistorySearch  map[string]string `json:"history_search"`
	WordNavigation map[string]string `json:"word_navigation"`
	LineNavigation map[string]string `json:"line_navigation"`
}

// VersionManager 版本管理器配置
type VersionManager struct {
	Enabled       bool                   `json:"enabled"`
	InitCommand   string                 `json:"init_command,omitempty"`
	EnvVars       map[string]interface{} `json:"env_vars,omitempty"`
	PathAdditions []string               `json:"path_additions,omitempty"`
	PostInstall   []string               `json:"post_install,omitempty"`
}

// GitTool Git 工具配置
type GitTool struct {
	Enabled    bool              `json:"enabled"`
	GitConfig  map[string]string `json:"git_config,omitempty"`
	Aliases    map[string]string `json:"aliases,omitempty"`
	Extensions []string          `json:"extensions,omitempty"`
}

// ExternalToolsConfig 外部工具配置
type ExternalToolsConfig struct {
	AutoInit map[string]string `json:"auto_init"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	MakeFlags         string `json:"makeflags"`
	AsyncLoading      bool   `json:"async_loading"`
	CompletionCache   bool   `json:"completion_cache"`
	PathDeduplication bool   `json:"path_deduplication"`
}

// PackagesConfig 包配置（从包文件加载）
type PackagesConfig struct {
	Categories map[string]Category `json:"categories"`
	Managers   map[string]Manager  `json:"package_managers"`
}

// Category 包分类
type Category struct {
	Description string                 `json:"description"`
	Priority    int                    `json:"priority"`
	Packages    map[string]PackageInfo `json:"packages"`
}

// PackageInfo 包信息
type PackageInfo struct {
	Description string            `json:"description"`
	Tags        []string          `json:"tags,omitempty"`
	Managers    map[string]string `json:"managers"` // 包管理器 -> 包名映射
	Optional    bool              `json:"optional,omitempty"`
	PostInstall []string          `json:"post_install,omitempty"`
}

// Manager 包管理器配置
type Manager struct {
	Command     string   `json:"command"`
	InstallArgs []string `json:"install_args"`
	Priority    int      `json:"priority"`
	Parallel    bool     `json:"parallel"`
}

// FunctionsConfig 函数配置（从 advanced_functions.json 加载）
type FunctionsConfig struct {
	Functions map[string]FunctionInfo `json:",inline"` // 直接展开所有函数
}

// FunctionInfo 单个函数信息
type FunctionInfo struct {
	Description string `json:"description"`
	Bash        string `json:"bash,omitempty"`
	Zsh         string `json:"zsh,omitempty"`
	PowerShell  string `json:"powershell,omitempty"`
}
