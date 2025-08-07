package platform

// PlatformInfo 包含平台相关信息
type PlatformInfo struct {
	OS           string    // 操作系统类型
	Architecture string    // 系统架构
	WSL          *WSLInfo  // WSL 信息（如果适用）
	PowerShell   *PSInfo   // PowerShell 信息（如果适用）
	Linux        *LinuxInfo // Linux 发行版信息（如果适用）
}

// WSLInfo WSL2 相关信息
type WSLInfo struct {
	IsWSL        bool   // 是否在 WSL 环境
	Version      string // WSL 版本 (1 或 2)
	Distribution string // Linux 发行版名称
	WindowsHome  string // Windows 用户目录路径
}

// PSInfo PowerShell 相关信息
type PSInfo struct {
	Version      string   // PowerShell 版本
	Edition      string   // 版本类型 (Desktop/Core)
	Platform     string   // 运行平台
	PSModulePath []string // 模块路径列表
	ExecutablePath string // 可执行文件路径
}

// LinuxInfo Linux 发行版信息
type LinuxInfo struct {
	Distribution string // 发行版名称 (arch, ubuntu, etc.)
	Version      string // 发行版版本
	PackageManager string // 默认包管理器
}