package template

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/bbq191/dotfiles-go/internal/config"
	"github.com/sirupsen/logrus"
)

// Engine 模板引擎，负责模板的加载、解析和渲染
type Engine struct {
	templates map[string]*template.Template // 已解析的模板缓存
	funcMap   template.FuncMap              // 全局模板函数映射
	logger    *logrus.Logger                // 日志记录器
	rootDir   string                       // 模板文件根目录
}

// NewEngine 创建新的模板引擎实例
func NewEngine(templateDir string, logger *logrus.Logger) *Engine {
	engine := &Engine{
		templates: make(map[string]*template.Template), // 初始化模板缓存
		logger:    logger,                             // 设置日志记录器
		rootDir:   templateDir,                        // 设置模板根目录
		funcMap:   createFuncMap(),                     // 创建函数映射表
	}

	return engine
}

// createFuncMap 创建全局模板函数映射表
func createFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap() // 加载 Sprig 标准函数库

	// 平台检测函数
	funcMap["isWindows"] = isWindows   // 检测 Windows 系统
	funcMap["isLinux"] = isLinux       // 检测 Linux 系统
	funcMap["osPath"] = osPath         // 转换路径格式
	funcMap["hasCommand"] = hasCommand // 检测命令可用性

	// 环境变量和路径处理
	funcMap["expandEnv"] = expandEnv    // 展开环境变量
	funcMap["pathJoin"] = filepath.Join // 连接路径
	funcMap["pathBase"] = filepath.Base // 获取文件名
	funcMap["pathDir"] = filepath.Dir   // 获取目录名

	// 字符串处理
	funcMap["quote"] = quote             // 添加引号
	funcMap["shellEscape"] = shellEscape // Shell 转义

	// 配置特定函数
	funcMap["keyBinding"] = keyBinding                           // 转换键绑定
	funcMap["shellName"] = shellName                             // 获取 shell 名称
	funcMap["getPlatformValue"] = getPlatformValue               // 获取平台值
	funcMap["formatFzfTheme"] = formatFzfTheme                   // 格式化 FZF 主题
	funcMap["generateFunctionComment"] = generateFunctionComment // 生成函数注释

	return funcMap
}

// createContextFuncMap 为特定模板上下文创建专用函数映射表
func (e *Engine) createContextFuncMap(context *TemplateContext) template.FuncMap {
	contextFuncMap := template.FuncMap{
		"isWSL": func() bool { // 检测 WSL 环境
			return context.Platform != nil && context.Platform.WSL != nil && context.Platform.WSL.IsWSL
		},
		"xdgPath": func(xdgType string, ctx *TemplateContext) string { // 获取 XDG 目录路径
			return xdgPath(xdgType, context)
		},
		"getActiveProxy": func() map[string]interface{} { // 获取激活代理配置
			return getActiveProxy(context)
		},
		"getVersionManagerEnv": func(vmConfig map[string]interface{}, envKey string) string { // 获取版本管理器环境变量
			return getVersionManagerEnv(vmConfig, envKey)
		},
	}

	return contextFuncMap
}

// LoadTemplates 批量加载指定类型的模板文件
func (e *Engine) LoadTemplates(templateTypes ...TemplateType) error {
	for _, templateType := range templateTypes { // 遍历所有模板类型
		if err := e.loadTemplate(templateType); err != nil { // 加载单个模板
			e.logger.Errorf("加载模板失败 %s: %v", templateType, err)
			return fmt.Errorf("加载模板 %s 失败: %w", templateType, err)
		}
	}
	return nil
}

// loadTemplate 加载并解析单个模板文件
func (e *Engine) loadTemplate(templateType TemplateType) error {
	templatePath := e.getTemplatePath(templateType)

	if _, err := os.Stat(templatePath); os.IsNotExist(err) { // 检查模板文件是否存在
		return fmt.Errorf("模板文件不存在: %s", templatePath)
	}

	// 创建扩展函数映射，包含占位符函数
	extendedFuncMap := e.funcMap
	extendedFuncMap["xdgPath"] = func(xdgType string, ctx *TemplateContext) string { return "" }
	extendedFuncMap["isWSL"] = func() bool { return false }
	extendedFuncMap["getActiveProxy"] = func() map[string]interface{} { return nil }
	extendedFuncMap["getVersionManagerEnv"] = func(vmConfig map[string]interface{}, envKey string) string { return "" }

	// 创建并解析模板
	tmpl, err := template.New(string(templateType)).
		Funcs(extendedFuncMap).
		ParseFiles(templatePath)

	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	e.templates[string(templateType)] = tmpl            // 存储解析后的模板
	e.logger.Debugf("模板加载成功: %s", templateType) // 记录成功日志

	return nil
}

// getTemplatePath 根据模板类型构建模板文件的完整路径
//
// 该方法将模板类型映射到实际的模板文件路径，支持预定义的模板类型
// 和动态路径构建。
//
// 参数：
// - templateType: 模板类型标识符
//
// 返回：
// - string: 模板文件的绝对路径
//
// 路径映射规则：
// - zsh -> templates/zsh/zshrc.tmpl
// - powershell -> templates/powershell/profile.ps1.tmpl
// - 其他 -> templates/{type}/{type}.tmpl
func (e *Engine) getTemplatePath(templateType TemplateType) string {
	switch templateType {
	case TemplateZsh:
		// ZSH 配置模板路径
		return filepath.Join(e.rootDir, "zsh", "zshrc.tmpl")
	case TemplatePowerShell:
		// PowerShell 配置模板路径
		return filepath.Join(e.rootDir, "powershell", "profile.ps1.tmpl")
	default:
		// 通用模板路径规则：templates/{类型}/{类型}.tmpl
		return filepath.Join(e.rootDir, string(templateType), string(templateType)+".tmpl")
	}
}

// Generate 使用指定模板和上下文数据生成配置文件
func (e *Engine) Generate(templateType TemplateType, context *TemplateContext, outputPath string) error {
	tmplKey := string(templateType)       // 转换模板类型为字符串键
	tmpl, exists := e.templates[tmplKey] // 检查模板是否已加载
	if !exists {
		if err := e.loadTemplate(templateType); err != nil { // 按需加载模板
			return err
		}
		tmpl = e.templates[tmplKey] // 获取已加载的模板
	}

	contextFuncMap := e.createContextFuncMap(context) // 创建上下文相关函数映射

	templatePath := e.getTemplatePath(templateType) // 获取模板文件路径
	allFuncMap := template.FuncMap{}                // 初始化完整函数映射

	// 合并全局函数和上下文函数
	for k, v := range e.funcMap {     // 复制全局函数
		allFuncMap[k] = v
	}
	for k, v := range contextFuncMap { // 复制上下文函数
		allFuncMap[k] = v
	}

	// 重新创建模板实例，绑定完整函数映射
	var parseErr error
	tmpl, parseErr = template.New(string(templateType)).
		Funcs(allFuncMap).
		ParseFiles(templatePath)
	if parseErr != nil {
		return fmt.Errorf("重新解析模板失败: %w", parseErr)
	}

	outputDir := filepath.Dir(outputPath)             // 获取输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil { // 创建输出目录
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	var buf bytes.Buffer // 创建缓冲区用于渲染输出
	
	// 执行模板渲染
	if err := tmpl.ExecuteTemplate(&buf, filepath.Base(templatePath), context); err != nil {
		return fmt.Errorf("模板执行失败: %w", err)
	}

	cleanedContent := e.cleanupEmptyLines(buf.String()) // 清理多余空行

	// 创建并写入输出文件
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outputFile.Close()

	if _, err := outputFile.WriteString(cleanedContent); err != nil { // 写入清理后内容
		return fmt.Errorf("写入文件失败: %w", err)
	}

	e.logger.Infof("配置文件生成成功: %s", outputPath)
	return nil
}

// ============================================================================
// 自定义模板函数实现
// ============================================================================
//
// 以下函数提供了模板系统的核心功能，包括平台检测、路径处理、
// 环境变量处理和配置特定的辅助功能。

// isWindows 检测当前运行环境是否为 Windows 系统
//
// 该函数用于模板中的平台特定配置生成，例如：
// {{if isWindows}}
//
//	# Windows 特定配置
//
// {{end}}
//
// 返回：
// - bool: true 表示运行在 Windows 系统上
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// isLinux 检测当前运行环境是否为 Linux 系统
//
// 该函数用于模板中的平台特定配置生成，例如：
// {{if isLinux}}
//
//	# Linux 特定配置
//
// {{end}}
//
// 返回：
// - bool: true 表示运行在 Linux 系统上
func isLinux() bool {
	return runtime.GOOS == "linux"
}

// osPath 将路径转换为当前操作系统的路径格式
//
// 在 Windows 系统上，将正斜杠转换为反斜杠；
// 在其他系统上保持原样。这对于生成跨平台兼容的路径非常有用。
//
// 参数：
// - path: 要转换的路径字符串
//
// 返回：
// - string: 转换后的系统特定路径格式
//
// 示例：
//
//	osPath("home/user/config")
//	// Windows: "home\\user\\config"
//	// Linux:   "home/user/config"
func osPath(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(path, "/", "\\")
	}
	return path
}

// xdgPath 根据 XDG 基础目录规范获取指定类型的目录路径
//
// XDG Base Directory Specification 定义了应用程序配置、数据、缓存
// 等文件的标准存储位置，有助于保持用户主目录的整洁。
//
// 参数：
// - xdgType: XDG 目录类型，支持的值：
//   - "config": XDG_CONFIG_HOME (配置文件目录)
//   - "data":   XDG_DATA_HOME (数据文件目录)
//   - "cache":  XDG_CACHE_HOME (缓存文件目录)
//   - "state":  XDG_STATE_HOME (状态文件目录)
//
// - ctx: 模板上下文，包含 XDG 配置信息
//
// 返回：
// - string: XDG 目录的绝对路径，如果 XDG 未启用或类型无效则返回空字符串
//
// 使用示例（在模板中）：
//
//	export XDG_CONFIG_HOME={{xdgPath "config" .}}
//	export XDG_DATA_HOME={{xdgPath "data" .}}
func xdgPath(xdgType string, ctx *TemplateContext) string {
	// 检查 XDG 配置是否启用
	if ctx.ZshConfig == nil || !ctx.ZshConfig.XDGDirectories.Enabled {
		return ""
	}

	// 根据请求的类型选择对应的路径配置
	var pathValue config.PathValue
	switch strings.ToLower(xdgType) {
	case "config":
		pathValue = ctx.ZshConfig.XDGDirectories.ConfigHome
	case "data":
		pathValue = ctx.ZshConfig.XDGDirectories.DataHome
	case "cache":
		pathValue = ctx.ZshConfig.XDGDirectories.CacheHome
	case "state":
		pathValue = ctx.ZshConfig.XDGDirectories.StateHome
	default:
		// 不支持的 XDG 类型
		return ""
	}

	// 根据当前运行平台获取相应的路径值
	platform := "linux"
	switch runtime.GOOS {
	case "windows":
		platform = "windows"
	case "darwin":
		platform = "macos"
	}

	return pathValue.Get(platform)
}

func expandEnv(input string) string {
	// 处理bash风格的默认值语法 ${VAR:-default}
	if strings.Contains(input, "${") && strings.Contains(input, ":-") {
		// 使用正则表达式匹配 ${VAR:-default} 模式
		re := regexp.MustCompile(`\$\{([^}:]+):-([^}]*)\}`)
		result := re.ReplaceAllStringFunc(input, func(match string) string {
			// 提取变量名和默认值
			parts := re.FindStringSubmatch(match)
			if len(parts) == 3 {
				varName := parts[1]
				defaultValue := parts[2]

				// 获取环境变量值
				if envValue := os.Getenv(varName); envValue != "" {
					return envValue
				}
				return defaultValue
			}
			return match
		})
		// 继续处理剩余的环境变量
		return os.ExpandEnv(result)
	}

	// 标准环境变量展开
	return os.ExpandEnv(input)
}

func quote(str string) string {
	return fmt.Sprintf("%q", str)
}

func shellEscape(str string) string {
	// 简单的shell转义
	if strings.ContainsAny(str, " \t\n\"'\\") {
		return fmt.Sprintf("'%s'", strings.ReplaceAll(str, "'", "'\"'\"'"))
	}
	return str
}

func hasCommand(cmd string) bool {
	// 检查命令是否存在于PATH中
	_, err := exec.LookPath(cmd)
	return err == nil
}

// keyBinding 将逻辑键名转换为实际的键绑定字符串
func keyBinding(keyName string) string {
	// 常用键绑定映射表
	keyMappings := map[string]string{
		// 方向键
		"up_arrow":    "^[[A",
		"down_arrow":  "^[[B",
		"right_arrow": "^[[C",
		"left_arrow":  "^[[D",

		// Ctrl组合键
		"ctrl_left":  "^[[1;5D",
		"ctrl_right": "^[[1;5C",
		"ctrl_up":    "^[[1;5A",
		"ctrl_down":  "^[[1;5B",

		// 功能键
		"home":      "^[[H",
		"end":       "^[[F",
		"delete":    "^[[3~",
		"page_up":   "^[[5~",
		"page_down": "^[[6~",
		"insert":    "^[[2~",

		// 特殊键
		"backspace": "^?",
		"tab":       "^I",
		"enter":     "^M",
		"escape":    "^[",
	}

	if mapping, exists := keyMappings[keyName]; exists {
		return mapping
	}

	// 如果没有找到映射，返回原始键名（可能是直接的键码）
	return keyName
}

// shellName 获取当前shell名称，用于替换模板中的{shell}占位符
func shellName() string {
	// 检测当前shell类型
	shell := os.Getenv("SHELL")
	if shell == "" {
		// 默认使用zsh（因为这主要用于zsh配置）
		return "zsh"
	}

	// 提取shell名称
	shellName := filepath.Base(shell)
	return shellName
}

// getPlatformValue 从PathValue对象中获取当前平台的值
func getPlatformValue(pathValue interface{}) string {
	// 处理不同类型的值
	switch v := pathValue.(type) {
	case string:
		// 直接返回字符串值，需要先展开环境变量
		return expandEnv(v)
	case config.PathValue:
		// 使用PathValue的Get方法
		platform := "linux"
		switch runtime.GOOS {
		case "windows":
			platform = "windows"
		case "darwin":
			platform = "macos"
		}
		result := v.Get(platform)
		return expandEnv(result)
	case map[string]interface{}:
		// 处理从JSON反序列化的map（版本管理器的平台特定配置）
		// 对于shell配置，优先使用zsh平台
		platformKeys := []string{"zsh", "bash"}
		if runtime.GOOS == "windows" {
			// Windows平台优先使用powershell配置
			platformKeys = []string{"powershell", "bash", "zsh"}
		}

		// 按优先级尝试不同的平台键
		for _, platform := range platformKeys {
			if val, exists := v[platform]; exists {
				if str, ok := val.(string); ok && str != "" {
					return expandEnv(str)
				}
			}
		}

		// 尝试获取默认值
		if val, exists := v["default"]; exists {
			if str, ok := val.(string); ok && str != "" {
				return expandEnv(str)
			}
		}

		// 最后尝试linux平台（向后兼容）
		if val, exists := v["linux"]; exists {
			if str, ok := val.(string); ok && str != "" {
				return expandEnv(str)
			}
		}
	}

	return ""
}

// formatFzfTheme 格式化FZF主题配置
func formatFzfTheme(theme interface{}) string {
	var parts []string

	switch t := theme.(type) {
	case map[string]string:
		for key, value := range t {
			if value != "" {
				parts = append(parts, fmt.Sprintf("--%s %s", key, value))
			}
		}
	case map[string]interface{}:
		for key, value := range t {
			if strValue, ok := value.(string); ok && strValue != "" {
				parts = append(parts, fmt.Sprintf("--%s %s", key, strValue))
			}
		}
	}

	return strings.Join(parts, " ")
}

// getActiveProxy 获取当前激活的代理配置
func getActiveProxy(context *TemplateContext) map[string]interface{} {
	if context.ZshConfig == nil || !context.ZshConfig.Proxy.Enabled {
		return nil
	}

	// 展开环境变量获取活跃的profile名称
	activeProfile := expandEnv(context.ZshConfig.Proxy.ActiveProfile)

	// 从类型化的Profiles中获取配置
	if profile, exists := context.ZshConfig.Proxy.Profiles[activeProfile]; exists {
		// 转换ProxyProfile为map[string]interface{}
		result := map[string]interface{}{
			"https_proxy": profile.HTTPSProxy,
			"http_proxy":  profile.HTTPProxy,
			"all_proxy":   profile.AllProxy,
			"no_proxy":    profile.NoProxy,
		}
		return result
	}

	return nil
}

// getVersionManagerEnv 获取版本管理器的环境变量值
func getVersionManagerEnv(vmConfig map[string]interface{}, envKey string) string {
	if envVars, exists := vmConfig["env_vars"].(map[string]interface{}); exists {
		if envValue, exists := envVars[envKey]; exists {
			return getPlatformValue(envValue)
		}
	}
	return ""
}

// GetTemplateNames 获取所有可用的模板名称
func (e *Engine) GetTemplateNames() []TemplateType {
	var templates []TemplateType

	// 扫描模板目录
	entries, err := os.ReadDir(e.rootDir)
	if err != nil {
		e.logger.Warnf("读取模板目录失败: %v", err)
		return []TemplateType{TemplateZsh, TemplatePowerShell} // 默认返回
	}

	for _, entry := range entries {
		if entry.IsDir() {
			templates = append(templates, TemplateType(entry.Name()))
		}
	}

	return templates
}

// generateFunctionComment 为函数生成详细的注释
func generateFunctionComment(funcName, funcCode string) string {
	// 定义每个函数的详细信息
	functionInfo := map[string]map[string]string{
		"mkcd": {
			"usage":   "mkcd <目录名>",
			"params":  "$1 - 要创建的目录名",
			"example": "mkcd my-new-project",
			"note":    "如果目录已存在，直接进入该目录",
		},
		"extract": {
			"usage":   "extract <压缩文件>",
			"params":  "$1 - 压缩文件路径",
			"example": "extract archive.tar.gz",
			"note":    "支持 .tar.gz, .zip, .7z, .rar 等多种格式",
		},
		"killport": {
			"usage":   "killport <端口号>",
			"params":  "$1 - 要终止进程的端口号",
			"example": "killport 3000",
			"note":    "强制终止占用指定端口的所有进程",
		},
		"serve": {
			"usage":   "serve [端口号]",
			"params":  "$1 - HTTP服务器端口号 (可选，默认8000)",
			"example": "serve 8080",
			"note":    "需要系统安装Python，用于快速启动HTTP文件服务器",
		},
		"weather": {
			"usage":   "weather [城市名]",
			"params":  "$1 - 城市名称 (可选，默认Beijing)",
			"example": "weather Shanghai",
			"note":    "获取指定城市的天气预报，支持中英文城市名",
		},
		"myip": {
			"usage":   "myip",
			"params":  "无参数",
			"example": "myip",
			"note":    "显示当前的公网IP地址",
		},
		"ports": {
			"usage":   "ports",
			"params":  "无参数",
			"example": "ports",
			"note":    "显示所有监听状态的网络端口",
		},
		"psmem": {
			"usage":   "psmem",
			"params":  "无参数",
			"example": "psmem",
			"note":    "显示内存占用最高的10个进程，优先使用procs工具",
		},
		"pscpu": {
			"usage":   "pscpu",
			"params":  "无参数",
			"example": "pscpu",
			"note":    "显示CPU占用最高的10个进程，优先使用procs工具",
		},
		"findfile": {
			"usage":   "findfile <文件名模式>",
			"params":  "$1 - 文件名或文件名模式",
			"example": "findfile config.json",
			"note":    "在当前目录递归查找文件，优先使用fd工具",
		},
		"finddir": {
			"usage":   "finddir <目录名模式>",
			"params":  "$1 - 目录名或目录名模式",
			"example": "finddir node_modules",
			"note":    "在当前目录递归查找目录，优先使用fd工具",
		},
		"backup": {
			"usage":   "backup <文件路径>",
			"params":  "$1 - 要备份的文件路径",
			"example": "backup ~/.zshrc",
			"note":    "创建带时间戳的文件备份",
		},
		"gitclean": {
			"usage":   "gitclean",
			"params":  "无参数",
			"example": "gitclean",
			"note":    "删除已合并到主分支的本地分支，保护main/master/develop分支",
		},
		"reload": {
			"usage":   "reload",
			"params":  "无参数",
			"example": "reload",
			"note":    "重新加载当前shell配置文件",
		},
		"sysinfo": {
			"usage":   "sysinfo",
			"params":  "无参数",
			"example": "sysinfo",
			"note":    "显示系统信息、内存使用情况和磁盘使用情况",
		},
	}

	info, exists := functionInfo[funcName]
	if !exists {
		// 如果没有预定义信息，返回基本注释
		return fmt.Sprintf("# 用法: %s", funcName)
	}

	var comment strings.Builder
	comment.WriteString(fmt.Sprintf("# 用法: %s\n", info["usage"]))
	comment.WriteString(fmt.Sprintf("# 参数: %s\n", info["params"]))
	comment.WriteString(fmt.Sprintf("# 示例: %s\n", info["example"]))
	if info["note"] != "" {
		comment.WriteString(fmt.Sprintf("# 说明: %s\n", info["note"]))
	}

	return strings.TrimSpace(comment.String())
}

// cleanupEmptyLines 清理模板生成内容中的多余空行
//
// 该方法会清理生成的配置文件中的连续空行，保持文件整洁：
// - 移除连续的多个空行，最多保留一个空行
// - 移除文件开头和结尾的多余空行
// - 保留单个空行用于分隔不同配置段落
//
// 参数：
// - content: 原始的模板渲染内容
//
// 返回：
// - string: 清理后的内容
//
// 处理策略：
// 1. 按行分割内容
// 2. 遍历每行，跟踪连续空行的数量
// 3. 当遇到连续空行时，最多保留一个
// 4. 去除文件首尾的多余空行
func (e *Engine) cleanupEmptyLines(content string) string {
	if content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	var result []string
	var consecutiveEmptyLines int

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		if trimmedLine == "" {
			// 当前行是空行
			consecutiveEmptyLines++
			
			// 只保留第一个空行，跳过后续连续的空行
			// 这样可以保留单个空行作为分隔符，但移除多个连续空行
			if consecutiveEmptyLines == 1 {
				result = append(result, line)
			}
		} else {
			// 当前行不是空行，重置计数器
			consecutiveEmptyLines = 0
			result = append(result, line)
		}
	}

	// 移除开头和结尾的空行
	cleanedResult := e.trimEmptyLinesFromEnds(result)

	return strings.Join(cleanedResult, "\n")
}

// trimEmptyLinesFromEnds 移除数组开头和结尾的空行
func (e *Engine) trimEmptyLinesFromEnds(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}

	// 移除开头的空行
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	// 移除结尾的空行
	end := len(lines) - 1
	for end >= start && strings.TrimSpace(lines[end]) == "" {
		end--
	}

	// 如果所有行都是空行，返回空数组
	if start > end {
		return []string{}
	}

	return lines[start : end+1]
}
