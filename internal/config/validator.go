package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// ConfigValidator 配置验证器
type ConfigValidator struct {
	validator *validator.Validate
	logger    *logrus.Logger
}

// NewConfigValidator 创建新的配置验证器
func NewConfigValidator(logger *logrus.Logger) *ConfigValidator {
	v := validator.New()
	cv := &ConfigValidator{
		validator: v,
		logger:    logger,
	}

	// 注册自定义验证规则
	cv.registerCustomValidators()

	return cv
}

// registerCustomValidators 注册自定义验证规则
func (cv *ConfigValidator) registerCustomValidators() {
	// 语义版本验证
	cv.validator.RegisterValidation("semver", cv.validateSemver)

	// 路径验证
	cv.validator.RegisterValidation("validpath", cv.validatePath)

	// 命令验证
	cv.validator.RegisterValidation("command", cv.validateCommand)

	// 环境变量名验证
	cv.validator.RegisterValidation("envvar", cv.validateEnvVar)

	// 代理URL验证
	cv.validator.RegisterValidation("proxyurl", cv.validateProxyURLField)

	// 包名验证
	cv.validator.RegisterValidation("packagename", cv.validatePackageName)
}

// ValidateConfig 验证完整配置
func (cv *ConfigValidator) ValidateConfig(config *DotfilesConfig) error {
	cv.logger.Debug("开始配置验证")

	// 结构体标签验证
	if err := cv.validator.Struct(config); err != nil {
		return cv.formatValidationError(err)
	}

	// 业务逻辑验证
	if err := cv.validateBusinessLogic(config); err != nil {
		return err
	}

	cv.logger.Debug("配置验证通过")
	return nil
}

// validateBusinessLogic 业务逻辑验证
func (cv *ConfigValidator) validateBusinessLogic(config *DotfilesConfig) error {
	// 验证用户配置
	if err := cv.validateUserConfig(config.User); err != nil {
		return fmt.Errorf("用户配置验证失败: %w", err)
	}

	// 验证路径配置
	if err := cv.validatePathsConfig(config.Paths); err != nil {
		return fmt.Errorf("路径配置验证失败: %w", err)
	}

	// 验证环境变量配置
	if err := cv.validateEnvironmentConfig(config.Environment); err != nil {
		return fmt.Errorf("环境变量配置验证失败: %w", err)
	}

	// 验证 Zsh 配置
	if config.ZshConfig != nil {
		if err := cv.validateZshConfig(config.ZshConfig); err != nil {
			return fmt.Errorf("ZSH 配置验证失败: %w", err)
		}
	}

	// 验证包配置
	if config.Packages != nil {
		if err := cv.validatePackagesConfig(config.Packages); err != nil {
			return fmt.Errorf("包配置验证失败: %w", err)
		}
	}

	return nil
}

// validateUserConfig 验证用户配置
func (cv *ConfigValidator) validateUserConfig(user UserConfig) error {
	// 验证邮箱格式（更严格的验证）
	if user.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(user.Email) {
			return fmt.Errorf("邮箱格式无效: %s", user.Email)
		}
	}

	// 验证编辑器是否可用
	if user.Editor != "" {
		if err := cv.validateEditorAvailability(user.Editor); err != nil {
			cv.logger.Warnf("编辑器 %s 可能不可用: %v", user.Editor, err)
		}
	}

	return nil
}

// validatePathsConfig 验证路径配置
func (cv *ConfigValidator) validatePathsConfig(paths PathsConfig) error {
	pathFields := map[string]PathValue{
		"projects":  paths.Projects,
		"dotfiles":  paths.Dotfiles,
		"scripts":   paths.Scripts,
		"templates": paths.Templates,
	}

	for name, pathValue := range pathFields {
		if err := cv.validatePathValue(name, pathValue); err != nil {
			return err
		}
	}

	return nil
}

// validatePathValue 验证路径值
func (cv *ConfigValidator) validatePathValue(name string, pv PathValue) error {
	// 检查是否为空
	if pv.Default == "" && len(pv.Platform) == 0 {
		if name == "projects" || name == "dotfiles" {
			return fmt.Errorf("必需的路径 %s 不能为空", name)
		}
		return nil // 可选路径可以为空
	}

	// 验证默认路径
	if pv.Default != "" {
		if err := cv.validateSinglePath(pv.Default); err != nil {
			return fmt.Errorf("路径 %s 的默认值无效: %w", name, err)
		}
	}

	// 验证平台特定路径
	for platform, path := range pv.Platform {
		if path != "" {
			if err := cv.validateSinglePath(path); err != nil {
				return fmt.Errorf("路径 %s 在平台 %s 上的值无效: %w", name, platform, err)
			}
		}
	}

	return nil
}

// validateSinglePath 验证单个路径
func (cv *ConfigValidator) validateSinglePath(path string) error {
	// 跳过包含环境变量的路径
	if strings.Contains(path, "$") || strings.Contains(path, "%") ||
		strings.Contains(path, "env:") {
		return nil
	}

	// 展开 ~ 符号
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("无法获取用户主目录: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// 检查路径格式是否有效（支持 Unix 和 Windows 路径）
	isValidPath := filepath.IsAbs(path) ||
		strings.HasPrefix(path, "~") ||
		strings.HasPrefix(path, ".") ||
		isWindowsPath(path)

	if !isValidPath {
		return fmt.Errorf("路径格式无效: %s", path)
	}

	return nil
}

// validateEnvironmentConfig 验证环境变量配置
func (cv *ConfigValidator) validateEnvironmentConfig(env map[string]string) error {
	for key, value := range env {
		// 验证环境变量名格式
		if !cv.isValidEnvVarName(key) {
			return fmt.Errorf("无效的环境变量名: %s", key)
		}

		// 验证特殊环境变量
		if err := cv.validateSpecialEnvVar(key, value); err != nil {
			return err
		}
	}

	return nil
}

// validateZshConfig 验证 Zsh 配置
func (cv *ConfigValidator) validateZshConfig(zshConfig *ZshIntegrationConfig) error {
	// 验证代理配置
	if err := cv.validateProxyConfig(zshConfig.Proxy); err != nil {
		return fmt.Errorf("代理配置验证失败: %w", err)
	}

	// 验证 XDG 配置
	if zshConfig.XDGDirectories.Enabled {
		if err := cv.validateXDGConfig(zshConfig.XDGDirectories); err != nil {
			return fmt.Errorf("XDG 配置验证失败: %w", err)
		}
	}

	// 验证版本管理器配置
	for name, vm := range zshConfig.VersionManagers {
		if err := cv.validateVersionManager(name, vm); err != nil {
			return fmt.Errorf("版本管理器 %s 配置验证失败: %w", name, err)
		}
	}

	return nil
}

// validateProxyConfig 验证代理配置
func (cv *ConfigValidator) validateProxyConfig(proxy ProxyConfig) error {
	if !proxy.Enabled {
		return nil
	}

	// 验证活动配置文件存在
	if proxy.ActiveProfile != "" && !strings.Contains(proxy.ActiveProfile, "$") {
		if _, exists := proxy.Profiles[proxy.ActiveProfile]; !exists {
			return fmt.Errorf("活动代理配置文件 %s 不存在", proxy.ActiveProfile)
		}
	}

	// 验证代理配置文件
	for name, profile := range proxy.Profiles {
		if err := cv.validateProxyProfile(name, profile); err != nil {
			return err
		}
	}

	return nil
}

// validateProxyProfile 验证代理配置文件
func (cv *ConfigValidator) validateProxyProfile(name string, profile ProxyProfile) error {
	proxyURLs := map[string]string{
		"http_proxy":  profile.HTTPProxy,
		"https_proxy": profile.HTTPSProxy,
		"all_proxy":   profile.AllProxy,
	}

	for urlType, url := range proxyURLs {
		if url != "" && !strings.Contains(url, "$") {
			if err := cv.validateProxyURL(url); err != nil {
				return fmt.Errorf("代理配置文件 %s 中的 %s 无效: %w", name, urlType, err)
			}
		}
	}

	return nil
}

// validateXDGConfig 验证 XDG 配置
func (cv *ConfigValidator) validateXDGConfig(xdg XDGConfig) error {
	xdgPaths := map[string]PathValue{
		"config_home": xdg.ConfigHome,
		"data_home":   xdg.DataHome,
		"state_home":  xdg.StateHome,
		"cache_home":  xdg.CacheHome,
		"runtime_dir": xdg.RuntimeDir,
		"user_bin":    xdg.UserBin,
	}

	for name, pathValue := range xdgPaths {
		if err := cv.validatePathValue("xdg."+name, pathValue); err != nil {
			return err
		}
	}

	return nil
}

// validateVersionManager 验证版本管理器配置
func (cv *ConfigValidator) validateVersionManager(name string, vm VersionManager) error {
	if !vm.Enabled {
		return nil
	}

	// 验证初始化命令
	if vm.InitCommand != "" {
		if err := cv.validateShellCommand(vm.InitCommand); err != nil {
			return fmt.Errorf("初始化命令无效: %w", err)
		}
	}

	// 验证环境变量
	for envKey, pathValue := range vm.EnvVars {
		if !cv.isValidEnvVarName(envKey) {
			return fmt.Errorf("无效的环境变量名: %s", envKey)
		}

		// 只对路径相关的环境变量进行路径验证
		if cv.isPathEnvironmentVariable(envKey) {
			if pathVal, ok := pathValue.(PathValue); ok {
				if err := cv.validatePathValue(fmt.Sprintf("%s.%s", name, envKey), pathVal); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// validatePackagesConfig 验证包配置
func (cv *ConfigValidator) validatePackagesConfig(packages *PackagesConfig) error {
	// 验证包管理器配置
	for name, manager := range packages.Managers {
		if err := cv.validatePackageManager(name, manager); err != nil {
			return err
		}
	}

	// 验证包分类
	for categoryName, category := range packages.Categories {
		if err := cv.validatePackageCategory(categoryName, category); err != nil {
			return err
		}
	}

	return nil
}

// validatePackageManager 验证包管理器配置
func (cv *ConfigValidator) validatePackageManager(name string, manager Manager) error {
	if manager.Command == "" {
		return fmt.Errorf("包管理器 %s 的命令不能为空", name)
	}

	if manager.Priority < 0 {
		return fmt.Errorf("包管理器 %s 的优先级不能为负数", name)
	}

	return nil
}

// validatePackageCategory 验证包分类
func (cv *ConfigValidator) validatePackageCategory(categoryName string, category Category) error {
	for packageName, packageInfo := range category.Packages {
		if err := cv.validatePackageInfo(packageName, packageInfo); err != nil {
			return fmt.Errorf("分类 %s 中的包 %s 验证失败: %w", categoryName, packageName, err)
		}
	}

	return nil
}

// validatePackageInfo 验证包信息
func (cv *ConfigValidator) validatePackageInfo(packageName string, info PackageInfo) error {
	if len(info.Managers) == 0 {
		return fmt.Errorf("包 %s 必须指定至少一个包管理器", packageName)
	}

	for manager, pkgName := range info.Managers {
		if pkgName == "" {
			return fmt.Errorf("包 %s 在管理器 %s 中的包名不能为空", packageName, manager)
		}
	}

	return nil
}

// 自定义验证函数
func (cv *ConfigValidator) validateSemver(fl validator.FieldLevel) bool {
	semverRegex := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	return semverRegex.MatchString(fl.Field().String())
}

func (cv *ConfigValidator) validatePath(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	return cv.validateSinglePath(path) == nil
}

func (cv *ConfigValidator) validateCommand(fl validator.FieldLevel) bool {
	command := fl.Field().String()
	return command != "" && !strings.ContainsAny(command, "\r\n")
}

func (cv *ConfigValidator) validateEnvVar(fl validator.FieldLevel) bool {
	envVar := fl.Field().String()
	return cv.isValidEnvVarName(envVar)
}

func (cv *ConfigValidator) validateProxyURL(url string) error {
	// 简单的代理 URL 格式验证
	proxyRegex := regexp.MustCompile(`^(http|https|socks4|socks5)://[^:]+:\d+$`)
	if !proxyRegex.MatchString(url) {
		return fmt.Errorf("代理 URL 格式无效: %s", url)
	}
	return nil
}

func (cv *ConfigValidator) validateProxyURLField(fl validator.FieldLevel) bool {
	url := fl.Field().String()
	return cv.validateProxyURL(url) == nil
}

func (cv *ConfigValidator) validatePackageName(fl validator.FieldLevel) bool {
	packageName := fl.Field().String()
	// 基本的包名验证
	return packageName != "" && !strings.ContainsAny(packageName, " \t\n\r")
}

// 辅助函数
func (cv *ConfigValidator) isValidEnvVarName(name string) bool {
	envVarRegex := regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	return envVarRegex.MatchString(name)
}

func (cv *ConfigValidator) validateEditorAvailability(editor string) error {
	// 提取命令名（去掉参数）
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("编辑器命令为空")
	}

	// command := parts[0]
	// 在这里可以添加更复杂的编辑器检测逻辑
	// 例如检查常见编辑器的存在性

	return nil
}

func (cv *ConfigValidator) validateSpecialEnvVar(key, value string) error {
	switch key {
	case "PATH":
		// 验证 PATH 环境变量格式
		if value != "" && !strings.Contains(value, ":") && !strings.Contains(value, ";") {
			cv.logger.Warnf("PATH 环境变量可能格式不正确: %s", value)
		}
	case "EDITOR":
		// 验证编辑器命令
		if value != "" {
			return cv.validateEditorAvailability(value)
		}
	}

	return nil
}

func (cv *ConfigValidator) validateShellCommand(command string) error {
	// 基本的 shell 命令验证
	if command == "" {
		return fmt.Errorf("命令不能为空")
	}

	// 检查危险字符
	dangerousChars := []string{";", "|", "&", ">", "<", "`", "$()"}
	for _, char := range dangerousChars {
		if strings.Contains(command, char) {
			cv.logger.Warnf("命令包含可能危险的字符: %s", command)
			break
		}
	}

	return nil
}

// formatValidationError 格式化验证错误
func (cv *ConfigValidator) formatValidationError(err error) error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return fmt.Errorf("验证错误格式异常: %w", err)
	}
	var messages []string

	for _, fieldErr := range validationErrors {
		fieldName := cv.getFieldDisplayName(fieldErr)

		switch fieldErr.Tag() {
		case "required":
			messages = append(messages, fmt.Sprintf("字段 %s 是必需的", fieldName))
		case "email":
			messages = append(messages, fmt.Sprintf("字段 %s 必须是有效的邮箱地址", fieldName))
		case "min":
			messages = append(messages, fmt.Sprintf("字段 %s 长度不能少于 %s", fieldName, fieldErr.Param()))
		case "semver":
			messages = append(messages, fmt.Sprintf("字段 %s 必须符合语义版本格式", fieldName))
		case "validpath":
			messages = append(messages, fmt.Sprintf("字段 %s 必须是有效的路径", fieldName))
		case "command":
			messages = append(messages, fmt.Sprintf("字段 %s 必须是有效的命令", fieldName))
		case "envvar":
			messages = append(messages, fmt.Sprintf("字段 %s 必须是有效的环境变量名", fieldName))
		case "proxyurl":
			messages = append(messages, fmt.Sprintf("字段 %s 必须是有效的代理 URL", fieldName))
		case "packagename":
			messages = append(messages, fmt.Sprintf("字段 %s 必须是有效的包名", fieldName))
		default:
			messages = append(messages, fmt.Sprintf("字段 %s 验证失败: %s", fieldName, fieldErr.Tag()))
		}
	}

	return fmt.Errorf("配置验证失败:\n  - %s", strings.Join(messages, "\n  - "))
}

// getFieldDisplayName 获取字段显示名称
func (cv *ConfigValidator) getFieldDisplayName(fieldErr validator.FieldError) string {
	// 可以在这里添加字段名称映射逻辑
	return fieldErr.Field()
}

// isWindowsPath 检查是否是 Windows 路径格式
func isWindowsPath(path string) bool {
	// Windows 路径格式: C:\path 或 D:\path
	windowsPathRegex := regexp.MustCompile(`^[A-Za-z]:[/\\]`)
	return windowsPathRegex.MatchString(path)
}

// isPathEnvironmentVariable 检查环境变量是否应该是路径类型
func (cv *ConfigValidator) isPathEnvironmentVariable(varName string) bool {
	// 常见的路径类型环境变量
	pathEnvVars := []string{
		"PATH", "HOME", "GOPATH", "JAVA_HOME", "PYTHONPATH",
		"NODE_PATH", "CARGO_HOME", "RUSTUP_HOME", "PYENV_ROOT",
		"FNM_DIR", "G_HOME", "JABBA_HOME", "PIPX_HOME", "PIPX_BIN_DIR",
		"PIPX_MAN_DIR", "PNPM_HOME", "NPM_CONFIG_USERCONFIG",
		"NPM_CONFIG_CACHE", "YARN_CACHE_FOLDER",
		"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME",
		"XDG_CACHE_HOME", "XDG_RUNTIME_DIR",
		"EDITOR_HOME", "MYSQL_HOME", "CLAUDE_CODE_GIT_BASH_PATH",
		"ZOXIDE_DATA_DIR",
	}

	// 检查是否以路径相关的后缀结尾
	pathSuffixes := []string{
		"_HOME", "_DIR", "_PATH", "_ROOT", "_BIN", "_MAN", "_CACHE",
		"_CONFIG", "_DATA", "_STATE", "_RUNTIME",
	}

	// 直接匹配
	for _, pathVar := range pathEnvVars {
		if varName == pathVar {
			return true
		}
	}

	// 后缀匹配
	for _, suffix := range pathSuffixes {
		if strings.HasSuffix(varName, suffix) {
			return true
		}
	}

	return false
}
