package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// YayManager Yay AUR包管理器实现
type YayManager struct {
	logger *logrus.Logger
}

// NewYayManager 创建Yay管理器实例
func NewYayManager(logger *logrus.Logger) *YayManager {
	return &YayManager{
		logger: logger,
	}
}

// Name 返回包管理器名称
func (y *YayManager) Name() string {
	return "yay"
}

// IsAvailable 检查yay是否可用
func (y *YayManager) IsAvailable() bool {
	// Yay 只在 Linux 上可用
	if runtime.GOOS != "linux" {
		y.logger.Debug("Yay 不适用于非Linux系统")
		return false
	}
	
	_, err := exec.LookPath("yay")
	available := err == nil
	y.logger.Debugf("Yay 可用性检查: %v", available)
	
	// 额外检查是否在Arch Linux系统上
	if available && !y.isArchLinux() {
		y.logger.Debug("Yay 可用但系统不是Arch Linux")
		return false
	}
	
	return available
}

// Install 安装包（支持AUR和官方仓库）
func (y *YayManager) Install(ctx context.Context, packageName string) error {
	y.logger.Infof("使用 Yay 安装包: %s", packageName)
	
	// 检查pacman数据库锁文件
	if err := y.checkPacmanLock(); err != nil {
		return err
	}
	
	// 检查sudo权限
	if err := y.checkSudoPermissions(); err != nil {
		return err
	}
	
	// 检查是否已安装
	if y.IsInstalled(packageName) {
		y.logger.Infof("包 %s 已安装，跳过", packageName)
		return nil
	}
	
	// 构建安装命令
	// yay -S --noconfirm --needed 包名
	args := []string{"-S", "--noconfirm", "--needed", packageName}
	cmd := exec.CommandContext(ctx, "yay", args...)
	
	y.logger.Debugf("执行命令: yay %s", strings.Join(args, " "))
	
	// 设置环境变量以防止交互提示
	cmd.Env = append(os.Environ(),
		"DEBIAN_FRONTEND=noninteractive",
		"LANG=C",
		"LC_ALL=C",
	)
	
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	
	// 总是显示命令输出以便调试
	if outputStr != "" {
		y.logger.Debugf("yay命令输出:\n%s", outputStr)
	}
	
	if err != nil {
		y.logger.Errorf("安装 %s 失败: %v", packageName, err)
		
		// 检查是否是权限问题
		if strings.Contains(outputStr, "sudo: a terminal is required") || 
		   strings.Contains(outputStr, "sudo: a password is required") ||
		   strings.Contains(outputStr, "error installing repo packages") {
			return fmt.Errorf("sudo权限验证失败，当前环境不支持密码输入\n\n💡 解决方案:\n1. 在真正的终端中运行此命令\n2. 或配置sudo无密码权限")
		}
		
		// 检查是否是锁文件问题
		if strings.Contains(outputStr, "db.lck") {
			return fmt.Errorf("pacman数据库被锁定，请运行 'sudo rm /var/lib/pacman/db.lck' 然后重试")
		}
		
		// 检查是否是网络问题
		if strings.Contains(outputStr, "failed to retrieve") || strings.Contains(outputStr, "download failed") {
			return fmt.Errorf("网络连接失败，请检查网络连接后重试: %v", err)
		}
		
		// 返回详细错误信息
		if outputStr != "" {
			return fmt.Errorf("安装失败: %v\n输出: %s", err, outputStr)
		}
		return fmt.Errorf("安装失败: %v", err)
	}
	
	y.logger.Infof("✅ 成功安装 %s", packageName)
	
	return nil
}

// IsInstalled 检查包是否已安装
func (y *YayManager) IsInstalled(packageName string) bool {
	// 使用 yay -Q 检查包是否已安装
	cmd := exec.Command("yay", "-Q", packageName)
	err := cmd.Run()
	
	installed := err == nil
	y.logger.Debugf("包 %s 安装状态: %v", packageName, installed)
	
	return installed
}

// Priority 返回优先级（高于pacman，因为yay可以处理官方仓库+AUR）
func (y *YayManager) Priority() int {
	return 0 // 最高优先级，优先于pacman
}

// SearchAUR 搜索AUR包
func (y *YayManager) SearchAUR(query string) ([]AURPackage, error) {
	cmd := exec.Command("yay", "-Ss", query)
	output, err := cmd.Output()
	
	if err != nil {
		return nil, err
	}
	
	packages := y.parseSearchOutput(string(output))
	return packages, nil
}

// IsFromAUR 检查包是否来自AUR
func (y *YayManager) IsFromAUR(packageName string) bool {
	cmd := exec.Command("yay", "-Si", packageName)
	output, err := cmd.Output()
	
	if err != nil {
		return false
	}
	
	// 检查输出中是否包含AUR相关信息
	outputStr := string(output)
	return strings.Contains(outputStr, "Repository") && 
		   (strings.Contains(outputStr, "aur") || strings.Contains(outputStr, "AUR"))
}

// GetPackageInfo 获取包详细信息
func (y *YayManager) GetPackageInfo(packageName string) (*AURPackageInfo, error) {
	cmd := exec.Command("yay", "-Si", packageName)
	output, err := cmd.Output()
	
	if err != nil {
		return nil, err
	}
	
	info := y.parsePackageInfo(string(output), packageName)
	return info, nil
}

// InstallFromAUR 专门从AUR安装包
func (y *YayManager) InstallFromAUR(ctx context.Context, packageName string, opts AURInstallOptions) error {
	y.logger.Infof("从AUR安装包: %s", packageName)
	
	args := []string{"-S", "--aur"}
	
	if opts.NoConfirm {
		args = append(args, "--noconfirm")
	}
	
	if opts.SkipReview {
		args = append(args, "--noconfirm") // 跳过PKGBUILD审查
	} else {
		y.logger.Warn("AUR包安装需要审查PKGBUILD，建议检查包源代码")
	}
	
	args = append(args, packageName)
	
	cmd := exec.CommandContext(ctx, "yay", args...)
	y.logger.Debugf("执行AUR安装命令: yay %s", strings.Join(args, " "))
	
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		y.logger.Errorf("从AUR安装 %s 失败: %v", packageName, err)
		y.logger.Debugf("AUR安装输出: %s", string(output))
		return err
	}
	
	y.logger.Infof("成功从AUR安装 %s", packageName)
	return nil
}

// isArchLinux 检查是否在Arch Linux系统上
func (y *YayManager) isArchLinux() bool {
	// 检查 /etc/os-release
	cmd := exec.Command("grep", "^ID=", "/etc/os-release")
	output, err := cmd.Output()
	
	if err != nil {
		return false
	}
	
	return strings.Contains(string(output), "arch")
}

// parseSearchOutput 解析搜索输出
func (y *YayManager) parseSearchOutput(output string) []AURPackage {
	packages := make([]AURPackage, 0)
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		// 解析包信息行
		if strings.Contains(line, "/") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				nameParts := strings.Split(parts[0], "/")
				if len(nameParts) == 2 {
					pkg := AURPackage{
						Repository:  nameParts[0],
						Name:        nameParts[1],
						Version:     parts[1],
						Description: strings.Join(parts[2:], " "),
					}
					packages = append(packages, pkg)
				}
			}
		}
	}
	
	return packages
}

// parsePackageInfo 解析包详细信息
func (y *YayManager) parsePackageInfo(output, packageName string) *AURPackageInfo {
	info := &AURPackageInfo{
		Name: packageName,
	}
	
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				switch key {
				case "Repository":
					info.Repository = value
				case "Version":
					info.Version = value
				case "Description":
					info.Description = value
				case "URL":
					info.URL = value
				case "Licenses":
					info.Licenses = strings.Split(value, " ")
				case "Depends On":
					if value != "None" {
						info.Dependencies = strings.Fields(value)
					}
				case "Make Deps":
					if value != "None" {
						info.MakeDependencies = strings.Fields(value)
					}
				case "Installed Size":
					info.InstalledSize = value
				}
			}
		}
	}
	
	return info
}

// checkPacmanLock 检查pacman数据库锁文件
func (y *YayManager) checkPacmanLock() error {
	lockFile := "/var/lib/pacman/db.lck"
	
	if _, err := os.Stat(lockFile); err == nil {
		y.logger.Warnf("检测到pacman数据库锁文件: %s", lockFile)
		return fmt.Errorf("pacman数据库被锁定，可能有其他包管理器正在运行\n\n💡 解决方案:\n1. 等待其他包管理器操作完成\n2. 如果确定没有其他进程，请运行: sudo rm %s\n3. 然后重试安装命令", lockFile)
	}
	
	return nil
}

// checkSudoPermissions 检查sudo权限
func (y *YayManager) checkSudoPermissions() error {
	// 测试sudo无密码权限
	cmd := exec.Command("sudo", "-n", "echo", "test")
	if err := cmd.Run(); err != nil {
		y.logger.Warnf("sudo权限检查失败: %v", err)
		return fmt.Errorf("yay需要sudo权限但当前环境无法提供密码验证\n\n💡 解决方案:\n1. 在真正的终端中运行此命令（推荐）\n2. 配置sudo无密码: 在/etc/sudoers中添加 '%s ALL=(ALL) NOPASSWD: /usr/bin/pacman'\n3. 使用系统包管理器而非yay", os.Getenv("USER"))
	}
	
	y.logger.Debugf("sudo权限检查通过")
	return nil
}

// AURPackage AUR包信息
type AURPackage struct {
	Repository  string `json:"repository"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// AURPackageInfo 详细的AUR包信息
type AURPackageInfo struct {
	Name             string   `json:"name"`
	Repository       string   `json:"repository"`
	Version          string   `json:"version"`
	Description      string   `json:"description"`
	URL              string   `json:"url"`
	Licenses         []string `json:"licenses"`
	Dependencies     []string `json:"dependencies"`
	MakeDependencies []string `json:"make_dependencies"`
	InstalledSize    string   `json:"installed_size"`
}

// AURInstallOptions AUR安装选项
type AURInstallOptions struct {
	NoConfirm  bool // 不要求确认
	SkipReview bool // 跳过PKGBUILD审查（有安全风险）
}