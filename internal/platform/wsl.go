package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetectWSL 检测 WSL 环境并返回详细信息
func DetectWSL() (*WSLInfo, error) {
	info := &WSLInfo{}
	
	// 方法1: 检查环境变量 WSL_DISTRO_NAME
	if distro := os.Getenv("WSL_DISTRO_NAME"); distro != "" {
		info.IsWSL = true
		info.Distribution = distro
	}
	
	// 方法2: 检查 /proc/version 文件
	if data, err := os.ReadFile("/proc/version"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
			info.IsWSL = true
			
			// 确定 WSL 版本
			if strings.Contains(content, "wsl2") {
				info.Version = "2"
			} else {
				info.Version = "1"
			}
		}
	}
	
	// 方法3: 检查 /proc/sys/kernel/osrelease
	if !info.IsWSL {
		if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
			content := strings.ToLower(string(data))
			if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
				info.IsWSL = true
				if strings.Contains(content, "wsl2") {
					info.Version = "2"
				} else {
					info.Version = "1"
				}
			}
		}
	}
	
	// 如果检测到 WSL，获取其他信息
	if info.IsWSL {
		if info.Distribution == "" {
			info.Distribution = detectWSLDistribution()
		}
		
		if windowsHome, err := getWindowsHome(); err == nil {
			info.WindowsHome = windowsHome
		}
	}
	
	return info, nil
}

// detectWSLDistribution 检测 WSL 发行版
func detectWSLDistribution() string {
	// 尝试从 /etc/os-release 获取发行版信息
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ID=") {
				return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
		}
	}
	
	// 备选方案：检查常见的发行版文件
	distributions := map[string]string{
		"/etc/arch-release":   "arch",
		"/etc/ubuntu-release": "ubuntu",
		"/etc/debian_version": "debian",
		"/etc/fedora-release": "fedora",
	}
	
	for file, distro := range distributions {
		if _, err := os.Stat(file); err == nil {
			return distro
		}
	}
	
	return "unknown"
}

// getWindowsHome 获取 Windows 用户主目录
func getWindowsHome() (string, error) {
	// 方法1: 使用 wslpath 命令
	if _, err := exec.LookPath("wslpath"); err == nil {
		// 获取 Windows 用户目录
		if output, err := exec.Command("cmd.exe", "/c", "echo %USERPROFILE%").Output(); err == nil {
			windowsPath := strings.TrimSpace(string(output))
			// 转换为 WSL 路径
			if unixPath, err := exec.Command("wslpath", "-u", windowsPath).Output(); err == nil {
				return strings.TrimSpace(string(unixPath)), nil
			}
		}
	}
	
	// 方法2: 通过环境变量计算
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("LOGNAME")
	}
	
	if username != "" {
		// 假设 C: 盘挂载在 /mnt/c
		return filepath.Join("/mnt/c/Users", username), nil
	}
	
	return "", os.ErrNotExist
}

// ConvertWSLPath 在 WSL 路径和 Windows 路径之间转换
func ConvertWSLPath(path string, toWindows bool) (string, error) {
	if _, err := exec.LookPath("wslpath"); err != nil {
		return "", err
	}
	
	var cmd *exec.Cmd
	if toWindows {
		cmd = exec.Command("wslpath", "-w", path)
	} else {
		cmd = exec.Command("wslpath", "-u", path)
	}
	
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

// IsWSL2 简单检查是否在 WSL2 环境中
func IsWSL2() bool {
	info, err := DetectWSL()
	if err != nil {
		return false
	}
	return info.IsWSL && info.Version == "2"
}

// IsWSL 简单检查是否在任何 WSL 环境中
func IsWSL() bool {
	info, err := DetectWSL()
	if err != nil {
		return false
	}
	return info.IsWSL
}