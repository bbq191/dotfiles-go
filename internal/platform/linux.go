package platform

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// DetectLinux 检测 Linux 发行版信息
func DetectLinux() (*LinuxInfo, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("not running on Linux")
	}
	
	info := &LinuxInfo{}
	
	// 检测发行版信息
	if err := info.detectDistribution(); err != nil {
		return nil, err
	}
	
	// 检测包管理器
	info.PackageManager = info.detectPackageManager()
	
	return info, nil
}

// detectDistribution 检测 Linux 发行版
func (info *LinuxInfo) detectDistribution() error {
	// 方法1: 读取 /etc/os-release
	if err := info.parseOSRelease(); err == nil {
		return nil
	}
	
	// 方法2: 检查特定发行版文件
	if err := info.detectFromReleaseFiles(); err == nil {
		return nil
	}
	
	// 方法3: 使用 lsb_release 命令
	if err := info.detectFromLSB(); err == nil {
		return nil
	}
	
	// 默认值
	info.Distribution = "unknown"
	info.Version = "unknown"
	
	return nil
}

// parseOSRelease 解析 /etc/os-release 文件
func (info *LinuxInfo) parseOSRelease() error {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		
		switch key {
		case "ID":
			info.Distribution = value
		case "VERSION_ID":
			info.Version = value
		case "NAME":
			if info.Distribution == "" {
				// 如果没有 ID，使用 NAME 作为备选
				info.Distribution = strings.ToLower(strings.Fields(value)[0])
			}
		}
	}
	
	if info.Distribution != "" {
		return nil
	}
	
	return fmt.Errorf("failed to parse distribution from os-release")
}

// detectFromReleaseFiles 从特定发行版文件检测
func (info *LinuxInfo) detectFromReleaseFiles() error {
	releaseFiles := map[string]string{
		"/etc/arch-release":     "arch",
		"/etc/ubuntu-release":   "ubuntu", 
		"/etc/debian_version":   "debian",
		"/etc/fedora-release":   "fedora",
		"/etc/centos-release":   "centos",
		"/etc/redhat-release":   "redhat",
		"/etc/opensuse-release": "opensuse",
		"/etc/alpine-release":   "alpine",
	}
	
	for file, distro := range releaseFiles {
		if _, err := os.Stat(file); err == nil {
			info.Distribution = distro
			
			// 尝试读取版本信息
			if data, err := os.ReadFile(file); err == nil {
				content := strings.TrimSpace(string(data))
				if content != "" {
					info.Version = extractVersion(content)
				}
			}
			
			return nil
		}
	}
	
	return fmt.Errorf("no known release files found")
}

// detectFromLSB 使用 lsb_release 命令检测
func (info *LinuxInfo) detectFromLSB() error {
	// 检测发行版 ID
	cmd := exec.Command("lsb_release", "-i", "-s")
	if output, err := cmd.Output(); err == nil {
		info.Distribution = strings.ToLower(strings.TrimSpace(string(output)))
	}
	
	// 检测版本
	cmd = exec.Command("lsb_release", "-r", "-s")
	if output, err := cmd.Output(); err == nil {
		info.Version = strings.TrimSpace(string(output))
	}
	
	if info.Distribution != "" {
		return nil
	}
	
	return fmt.Errorf("lsb_release command failed or not available")
}

// detectPackageManager 检测默认包管理器
func (info *LinuxInfo) detectPackageManager() string {
	packageManagers := map[string][]string{
		"pacman": {"pacman"},
		"apt":    {"apt", "apt-get"},
		"yum":    {"yum"},
		"dnf":    {"dnf"},
		"zypper": {"zypper"},
		"apk":    {"apk"},
		"portage": {"emerge"},
	}
	
	// 基于发行版推断
	switch info.Distribution {
	case "arch", "archlinux", "manjaro":
		return "pacman"
	case "ubuntu", "debian", "linuxmint":
		return "apt"
	case "fedora":
		return "dnf"
	case "centos", "rhel", "redhat":
		return "yum"
	case "opensuse", "suse":
		return "zypper"
	case "alpine":
		return "apk"
	case "gentoo":
		return "portage"
	}
	
	// 通过命令存在性检测
	for manager, commands := range packageManagers {
		for _, cmd := range commands {
			if _, err := exec.LookPath(cmd); err == nil {
				return manager
			}
		}
	}
	
	return "unknown"
}

// extractVersion 从发行版文件内容中提取版本号
func extractVersion(content string) string {
	// 简单的版本提取逻辑
	fields := strings.Fields(content)
	for _, field := range fields {
		// 查找包含数字和点的字段
		if strings.ContainsAny(field, "0123456789.") {
			return field
		}
	}
	
	return "unknown"
}

// IsArch 检查是否为 Arch Linux
func (info *LinuxInfo) IsArch() bool {
	return info.Distribution == "arch" || info.Distribution == "archlinux"
}

// IsDebian 检查是否为 Debian 系发行版
func (info *LinuxInfo) IsDebian() bool {
	debianDistros := []string{"debian", "ubuntu", "linuxmint", "elementary"}
	for _, distro := range debianDistros {
		if info.Distribution == distro {
			return true
		}
	}
	return false
}

// IsRedHat 检查是否为 Red Hat 系发行版
func (info *LinuxInfo) IsRedHat() bool {
	redhatDistros := []string{"fedora", "centos", "rhel", "redhat"}
	for _, distro := range redhatDistros {
		if info.Distribution == distro {
			return true
		}
	}
	return false
}

// HasPackageManager 检查是否有指定的包管理器
func HasPackageManager(manager string) bool {
	_, err := exec.LookPath(manager)
	return err == nil
}

// GetAvailablePackageManagers 获取系统中可用的包管理器列表
func GetAvailablePackageManagers() []string {
	managers := []string{"pacman", "apt", "apt-get", "yum", "dnf", "zypper", "apk", "emerge", "yay", "paru"}
	var available []string
	
	for _, manager := range managers {
		if HasPackageManager(manager) {
			available = append(available, manager)
		}
	}
	
	return available
}