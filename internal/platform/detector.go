package platform

import (
	"fmt"
	"os"
	"runtime"
)

// Detector 平台检测器
type Detector struct{}

// NewDetector 创建新的平台检测器
func NewDetector() *Detector {
	return &Detector{}
}

// DetectPlatform 检测当前平台的完整信息
func (d *Detector) DetectPlatform() (*PlatformInfo, error) {
	info := &PlatformInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
	
	// 检测 WSL 环境
	if wslInfo, err := DetectWSL(); err == nil && wslInfo.IsWSL {
		info.WSL = wslInfo
	}
	
	// 检测 PowerShell
	if psInfo, err := DetectPowerShell(); err == nil {
		info.PowerShell = psInfo
	}
	
	// 检测 Linux 发行版信息
	if runtime.GOOS == "linux" {
		if linuxInfo, err := DetectLinux(); err == nil {
			info.Linux = linuxInfo
		}
	}
	
	return info, nil
}

// String 返回平台信息的字符串表示
func (info *PlatformInfo) String() string {
	str := fmt.Sprintf("Platform: %s/%s", info.OS, info.Architecture)
	
	if info.WSL != nil && info.WSL.IsWSL {
		str += fmt.Sprintf("\nWSL: %s v%s (%s)", info.WSL.Distribution, info.WSL.Version, "Windows Subsystem for Linux")
		if info.WSL.WindowsHome != "" {
			str += fmt.Sprintf("\nWindows Home: %s", info.WSL.WindowsHome)
		}
	}
	
	if info.Linux != nil {
		str += fmt.Sprintf("\nLinux: %s %s (Package Manager: %s)", 
			info.Linux.Distribution, info.Linux.Version, info.Linux.PackageManager)
	}
	
	if info.PowerShell != nil {
		str += fmt.Sprintf("\nPowerShell: %s %s (%s)", 
			info.PowerShell.Version, info.PowerShell.Edition, info.PowerShell.ExecutablePath)
	}
	
	return str
}

// IsWSLEnvironment 检查是否在 WSL 环境中
func (info *PlatformInfo) IsWSLEnvironment() bool {
	return info.WSL != nil && info.WSL.IsWSL
}

// IsWSL2Environment 检查是否在 WSL2 环境中
func (info *PlatformInfo) IsWSL2Environment() bool {
	return info.WSL != nil && info.WSL.IsWSL && info.WSL.Version == "2"
}

// SupportsPowerShell 检查是否支持 PowerShell
func (info *PlatformInfo) SupportsPowerShell() bool {
	return info.PowerShell != nil
}

// SupportsPackageManager 检查是否支持指定的包管理器
func (info *PlatformInfo) SupportsPackageManager(manager string) bool {
	switch manager {
	case "pacman", "yay", "paru":
		return info.Linux != nil && info.Linux.IsArch()
	case "apt", "apt-get":
		return info.Linux != nil && info.Linux.IsDebian()
	case "yum", "dnf":
		return info.Linux != nil && info.Linux.IsRedHat()
	case "winget", "scoop", "choco":
		return info.OS == "windows" || info.IsWSLEnvironment()
	default:
		return HasPackageManager(manager)
	}
}

// GetRecommendedPackageManagers 获取推荐的包管理器列表
func (info *PlatformInfo) GetRecommendedPackageManagers() []string {
	var managers []string
	
	if info.IsWSLEnvironment() {
		// WSL 环境下推荐 Linux 包管理器
		if info.Linux != nil {
			switch info.Linux.PackageManager {
			case "pacman":
				managers = append(managers, "yay", "pacman") // AUR helper 优先
			case "apt":
				managers = append(managers, "apt")
			}
		}
		
		// 也可以使用 Windows 包管理器（通过 WSL 互操作）
		if HasPackageManager("winget.exe") {
			managers = append(managers, "winget")
		}
	} else if info.OS == "windows" {
		// 纯 Windows 环境
		if HasPackageManager("winget") {
			managers = append(managers, "winget")
		}
		if HasPackageManager("scoop") {
			managers = append(managers, "scoop")
		}
		if HasPackageManager("choco") {
			managers = append(managers, "choco")
		}
	} else if info.OS == "linux" && info.Linux != nil {
		// 纯 Linux 环境
		switch info.Linux.PackageManager {
		case "pacman":
			if HasPackageManager("yay") {
				managers = append(managers, "yay")
			}
			if HasPackageManager("paru") {
				managers = append(managers, "paru")
			}
			managers = append(managers, "pacman")
		default:
			managers = append(managers, info.Linux.PackageManager)
		}
	}
	
	return managers
}

// GetConfigPaths 获取配置文件路径建议
func (info *PlatformInfo) GetConfigPaths() map[string]string {
	paths := make(map[string]string)
	
	if info.IsWSLEnvironment() {
		// WSL 环境路径
		paths["home"] = "/home/" + getUsername()
		paths["config"] = paths["home"] + "/.config"
		if info.WSL.WindowsHome != "" {
			paths["windows_home"] = info.WSL.WindowsHome
		}
	} else if info.OS == "windows" {
		// Windows 环境路径
		paths["home"] = "%USERPROFILE%"
		paths["config"] = "%APPDATA%"
		paths["local_config"] = "%LOCALAPPDATA%"
	} else {
		// Unix 环境路径
		paths["home"] = "~"
		paths["config"] = "~/.config"
		paths["local"] = "~/.local"
	}
	
	return paths
}

// getUsername 获取当前用户名
func getUsername() string {
	if username := getEnvVar("USER"); username != "" {
		return username
	}
	if username := getEnvVar("LOGNAME"); username != "" {
		return username
	}
	if username := getEnvVar("USERNAME"); username != "" {
		return username
	}
	return "user"
}

// getEnvVar 安全获取环境变量
func getEnvVar(key string) string {
	return os.Getenv(key)
}