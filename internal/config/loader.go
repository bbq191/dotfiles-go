package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bbq191/dotfiles-go/internal/platform"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
	configDir   string
	platform    string
	detector    *platform.Detector
	validator   *validator.Validate
	logger      *logrus.Logger
}

// NewConfigLoader 创建新的配置加载器
func NewConfigLoader(configDir string, logger *logrus.Logger) *ConfigLoader {
	return &ConfigLoader{
		configDir: configDir,
		platform:  detectCurrentPlatform(),
		detector:  platform.NewDetector(),
		validator: validator.New(),
		logger:    logger,
	}
}

// LoadConfig 加载完整配置
func (cl *ConfigLoader) LoadConfig() (*DotfilesConfig, error) {
	cl.logger.Debug("开始加载配置文件")

	// 加载主配置文件
	config, err := cl.loadMainConfig()
	if err != nil {
		return nil, fmt.Errorf("加载主配置失败: %w", err)
	}

	// 加载 Zsh 集成配置
	if zshConfig, err := cl.loadZshConfig(); err == nil {
		config.ZshConfig = zshConfig
		cl.logger.Debug("已加载 Zsh 集成配置")
	} else {
		cl.logger.Warnf("加载 Zsh 配置失败: %v", err)
	}

	// 加载包配置
	if packagesConfig, err := cl.loadPackagesConfig(); err == nil {
		config.Packages = packagesConfig
		cl.logger.Debug("已加载包配置")
	} else {
		cl.logger.Warnf("加载包配置失败: %v", err)
	}

	// 加载函数配置
	if functionsConfig, err := cl.loadFunctionsConfig(); err == nil {
		config.Functions = functionsConfig
		cl.logger.Debug("已加载函数配置")
	} else {
		cl.logger.Warnf("加载函数配置失败: %v", err)
	}

	// 验证配置
	if err := cl.validateConfig(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 处理配置后处理（环境变量展开等）
	cl.postProcessConfig(config)

	cl.logger.Info("配置加载完成")
	return config, nil
}

// loadMainConfig 加载主配置文件
func (cl *ConfigLoader) loadMainConfig() (*DotfilesConfig, error) {
	// 直接读取 JSON 文件
	configPath := filepath.Join(cl.configDir, "shared.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	cl.logger.Debugf("原始配置键: %v", getMapKeys(rawConfig))

	// 手动构建配置结构
	config := &DotfilesConfig{}

	// 解析用户配置
	if userData, ok := rawConfig["user"]; ok {
		if userDataMap, ok := userData.(map[string]interface{}); ok {
			config.User = UserConfig{
				Name:    getStringFromMap(userDataMap, "name"),
				Email:   getStringFromMap(userDataMap, "email"),
				Editor:  getStringFromMap(userDataMap, "editor"),
				Browser: getStringFromMap(userDataMap, "browser"),
			}
			cl.logger.Debugf("解析用户配置: Name=%s, Email=%s", config.User.Name, config.User.Email)
		} else {
			cl.logger.Warnf("用户数据不是 map 类型: %T", userData)
		}
	} else {
		cl.logger.Warn("配置中未找到 user 字段")
	}

	// 解析路径配置
	if pathsData, ok := rawConfig["paths"]; ok {
		if pathsData, ok := pathsData.(map[string]interface{}); ok {
			config.Paths = PathsConfig{
				Projects:  cl.parsePathValue(pathsData["projects"]),
				Dotfiles:  cl.parsePathValue(pathsData["dotfiles"]),
				Scripts:   cl.parsePathValue(pathsData["scripts"]),
				Templates: cl.parsePathValue(pathsData["templates"]),
			}
		}
	}

	// 解析环境变量
	if envData, ok := rawConfig["environment"]; ok {
		if envData, ok := envData.(map[string]interface{}); ok {
			config.Environment = make(map[string]string)
			for k, v := range envData {
				if strVal, ok := v.(string); ok {
					config.Environment[k] = strVal
				}
			}
		}
	}

	// 解析功能配置
	if featuresData, ok := rawConfig["features"]; ok {
		if featuresData, ok := featuresData.(map[string]interface{}); ok {
			config.Features = FeaturesConfig{
				GitIntegration:   getBoolFromMap(featuresData, "git_integration"),
				NodejsManagement: getBoolFromMap(featuresData, "nodejs_management"),
				PythonManagement: getBoolFromMap(featuresData, "python_management"),
			}
		}
	}

	// 设置版本
	if version, ok := rawConfig["version"]; ok {
		if versionStr, ok := version.(string); ok {
			config.Version = versionStr
		}
	}

	cl.logger.Debugf("已加载主配置文件: %s", configPath)
	return config, nil
}

// loadZshConfig 加载 Zsh 集成配置
func (cl *ConfigLoader) loadZshConfig() (*ZshIntegrationConfig, error) {
	configPath := filepath.Join(cl.configDir, "zsh_integration.json")
	return loadJSONConfig[ZshIntegrationConfig](configPath)
}

// loadPackagesConfig 加载包配置
func (cl *ConfigLoader) loadPackagesConfig() (*PackagesConfig, error) {
	// 尝试加载平台特定的包配置
	platformFiles := []string{
		fmt.Sprintf("packages/%s.json", cl.platform),
		"packages/linux.json", // 备选
		"packages/arch.json",  // 备选
	}

	for _, filename := range platformFiles {
		configPath := filepath.Join(cl.configDir, filename)
		if _, err := os.Stat(configPath); err == nil {
			cl.logger.Debugf("尝试加载包配置: %s", configPath)
			if config, err := loadJSONConfig[PackagesConfig](configPath); err == nil {
				return config, nil
			}
		}
	}

	return nil, fmt.Errorf("未找到适合的包配置文件")
}

// loadFunctionsConfig 加载函数配置
func (cl *ConfigLoader) loadFunctionsConfig() (*FunctionsConfig, error) {
	configPath := filepath.Join(cl.configDir, "advanced_functions.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var functions map[string]FunctionInfo
	if err := json.Unmarshal(data, &functions); err != nil {
		return nil, fmt.Errorf("解析函数配置文件失败: %w", err)
	}

	return &FunctionsConfig{
		Functions: functions,
	}, nil
}

// loadJSONConfig 通用 JSON 配置加载器
func loadJSONConfig[T any](configPath string) (*T, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config T
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件 %s 失败: %w", configPath, err)
	}

	return &config, nil
}

// validateConfig 验证配置
func (cl *ConfigLoader) validateConfig(config *DotfilesConfig) error {
	cl.logger.Debug("开始验证配置")

	if err := cl.validator.Struct(config); err != nil {
		return cl.formatValidationError(err)
	}

	// 自定义验证逻辑
	if err := cl.customValidation(config); err != nil {
		return err
	}

	cl.logger.Debug("配置验证通过")
	return nil
}

// formatValidationError 格式化验证错误
func (cl *ConfigLoader) formatValidationError(err error) error {
	validationErrors := err.(validator.ValidationErrors)
	var messages []string

	for _, fieldErr := range validationErrors {
		switch fieldErr.Tag() {
		case "required":
			messages = append(messages, fmt.Sprintf("字段 %s 是必需的", fieldErr.Field()))
		case "email":
			messages = append(messages, fmt.Sprintf("字段 %s 必须是有效的邮箱地址", fieldErr.Field()))
		case "min":
			messages = append(messages, fmt.Sprintf("字段 %s 长度不能少于 %s", fieldErr.Field(), fieldErr.Param()))
		case "semver":
			messages = append(messages, fmt.Sprintf("字段 %s 必须符合语义版本格式", fieldErr.Field()))
		default:
			messages = append(messages, fmt.Sprintf("字段 %s 验证失败: %s", fieldErr.Field(), fieldErr.Tag()))
		}
	}

	return fmt.Errorf("配置验证失败:\n  - %s", strings.Join(messages, "\n  - "))
}

// customValidation 自定义验证逻辑
func (cl *ConfigLoader) customValidation(config *DotfilesConfig) error {
	// 验证用户邮箱格式
	if config.User.Email != "" && !strings.Contains(config.User.Email, "@") {
		return fmt.Errorf("用户邮箱格式无效: %s", config.User.Email)
	}

	// 验证路径配置
	if err := cl.validatePaths(config.Paths); err != nil {
		return fmt.Errorf("路径配置验证失败: %w", err)
	}

	return nil
}

// validatePaths 验证路径配置
func (cl *ConfigLoader) validatePaths(paths PathsConfig) error {
	pathFields := map[string]PathValue{
		"projects": paths.Projects,
		"dotfiles": paths.Dotfiles,
	}

	for name, pathValue := range pathFields {
		if pathValue.Default == "" && len(pathValue.Platform) == 0 {
			return fmt.Errorf("路径 %s 不能为空", name)
		}
	}

	return nil
}

// postProcessConfig 配置后处理
func (cl *ConfigLoader) postProcessConfig(config *DotfilesConfig) {
	cl.logger.Debug("开始配置后处理")

	// 展开环境变量
	cl.expandEnvironmentVariables(config)

	// 设置默认值
	cl.setDefaultValues(config)

	cl.logger.Debug("配置后处理完成")
}

// expandEnvironmentVariables 展开环境变量
func (cl *ConfigLoader) expandEnvironmentVariables(config *DotfilesConfig) {
	// 展开路径中的环境变量
	config.Paths.Projects = cl.expandPathValue(config.Paths.Projects)
	config.Paths.Dotfiles = cl.expandPathValue(config.Paths.Dotfiles)
	config.Paths.Scripts = cl.expandPathValue(config.Paths.Scripts)
	config.Paths.Templates = cl.expandPathValue(config.Paths.Templates)

	// 展开环境变量配置
	for key, value := range config.Environment {
		config.Environment[key] = cl.expandEnvVars(value)
	}

	// 展开 Zsh 配置中的环境变量
	if config.ZshConfig != nil {
		cl.expandZshConfigVariables(config.ZshConfig)
	}
}

// expandPathValue 展开路径值中的环境变量
func (cl *ConfigLoader) expandPathValue(pv PathValue) PathValue {
	if pv.Default != "" {
		pv.Default = cl.expandEnvVars(pv.Default)
	}

	if pv.Platform != nil {
		expanded := make(map[string]string)
		for platform, path := range pv.Platform {
			expanded[platform] = cl.expandEnvVars(path)
		}
		pv.Platform = expanded
	}

	return pv
}

// expandEnvVars 展开环境变量，支持多种格式
func (cl *ConfigLoader) expandEnvVars(s string) string {
	// 先处理 PowerShell 格式的环境变量 $env:VARNAME
	psEnvRegex := regexp.MustCompile(`\$env:([A-Za-z_][A-Za-z0-9_]*)`)
	s = psEnvRegex.ReplaceAllStringFunc(s, func(match string) string {
		varName := psEnvRegex.FindStringSubmatch(match)[1]
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // 如果环境变量不存在，保持原样
	})

	// 然后使用标准的 os.ExpandEnv 处理其他格式
	return os.ExpandEnv(s)
}

// expandZshConfigVariables 展开 Zsh 配置中的环境变量
func (cl *ConfigLoader) expandZshConfigVariables(zshConfig *ZshIntegrationConfig) {
	// 展开 XDG 目录配置
	if zshConfig.XDGDirectories.Enabled {
		zshConfig.XDGDirectories.ConfigHome = cl.expandPathValue(zshConfig.XDGDirectories.ConfigHome)
		zshConfig.XDGDirectories.DataHome = cl.expandPathValue(zshConfig.XDGDirectories.DataHome)
		zshConfig.XDGDirectories.StateHome = cl.expandPathValue(zshConfig.XDGDirectories.StateHome)
		zshConfig.XDGDirectories.CacheHome = cl.expandPathValue(zshConfig.XDGDirectories.CacheHome)
		zshConfig.XDGDirectories.RuntimeDir = cl.expandPathValue(zshConfig.XDGDirectories.RuntimeDir)
		zshConfig.XDGDirectories.UserBin = cl.expandPathValue(zshConfig.XDGDirectories.UserBin)
	}

	// 展开版本管理器配置
	for name, vm := range zshConfig.VersionManagers {
		if vm.EnvVars != nil {
			expanded := make(map[string]interface{})
			for key, pathValue := range vm.EnvVars {
				if pathVal, ok := pathValue.(PathValue); ok {
					expanded[key] = cl.expandPathValue(pathVal)
				} else {
					expanded[key] = pathValue
				}
			}
			vm.EnvVars = expanded
			zshConfig.VersionManagers[name] = vm
		}
	}

	// 展开开发环境配置
	for envName, envConfig := range zshConfig.DevelopmentEnvironments {
		expanded := make(map[string]PathValue)
		for key, pathValue := range envConfig {
			expanded[key] = cl.expandPathValue(pathValue)
		}
		zshConfig.DevelopmentEnvironments[envName] = expanded
	}
}

// setDefaultValues 设置默认值
func (cl *ConfigLoader) setDefaultValues(config *DotfilesConfig) {
	// 设置版本默认值
	if config.Version == "" {
		config.Version = "1.0.0"
	}

	// 设置默认编辑器
	if config.User.Editor == "" {
		if editor := os.Getenv("EDITOR"); editor != "" {
			config.User.Editor = editor
		} else {
			config.User.Editor = "nano"
		}
	}

	// 初始化环境变量映射
	if config.Environment == nil {
		config.Environment = make(map[string]string)
	}
}

// detectCurrentPlatform 检测当前平台
func detectCurrentPlatform() string {
	detector := platform.NewDetector()
	info, err := detector.DetectPlatform()
	if err != nil {
		return "linux" // 默认值
	}

	if info.IsWSLEnvironment() {
		return "wsl"
	}

	if info.Linux != nil && info.Linux.IsArch() {
		return "arch"
	}

	return info.OS
}

// GetConfigDir 获取配置目录路径
func GetConfigDir() string {
	// 优先检查环境变量
	if configDir := os.Getenv("DOTFILES_CONFIG_DIR"); configDir != "" {
		return configDir
	}

	// 检查当前工作目录下的 configs 文件夹
	if cwd, err := os.Getwd(); err == nil {
		configPath := filepath.Join(cwd, "configs")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// 检查用户家目录下的 .config/dotfiles
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "dotfiles")
	}

	return "configs" // 默认值
}



// 辅助解析函数
func getStringFromMap(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

func getBoolFromMap(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

func (cl *ConfigLoader) parsePathValue(data interface{}) PathValue {
	if data == nil {
		return PathValue{}
	}

	if strVal, ok := data.(string); ok {
		return PathValue{
			Default:  strVal,
			Platform: nil,
		}
	}

	if mapData, ok := data.(map[string]interface{}); ok {
		platformMap := make(map[string]string)
		for k, v := range mapData {
			if strVal, ok := v.(string); ok {
				platformMap[k] = strVal
			}
		}
		return PathValue{
			Default:  "",
			Platform: platformMap,
		}
	}

	return PathValue{}
}

func getMapKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}