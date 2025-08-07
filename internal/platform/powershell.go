package platform

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// DetectPowerShell 检测 PowerShell 环境并返回详细信息
func DetectPowerShell() (*PSInfo, error) {
	info := &PSInfo{}
	
	// 确定 PowerShell 可执行文件路径
	psExe, err := findPowerShellExecutable()
	if err != nil {
		return nil, fmt.Errorf("PowerShell not found: %w", err)
	}
	
	info.ExecutablePath = psExe
	info.Platform = runtime.GOOS
	
	// 获取 PowerShell 版本信息
	if err := info.getVersionInfo(psExe); err != nil {
		return nil, fmt.Errorf("failed to get PowerShell version: %w", err)
	}
	
	// 获取模块路径
	if err := info.getModulePaths(psExe); err != nil {
		// 模块路径获取失败不是致命错误，记录日志但继续
		info.PSModulePath = []string{}
	}
	
	return info, nil
}

// findPowerShellExecutable 查找 PowerShell 可执行文件
func findPowerShellExecutable() (string, error) {
	// 优先查找 PowerShell Core (pwsh)
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path, nil
	}
	
	// 在 Windows 上查找 Windows PowerShell
	if runtime.GOOS == "windows" {
		if path, err := exec.LookPath("powershell"); err == nil {
			return path, nil
		}
	}
	
	return "", fmt.Errorf("no PowerShell executable found")
}

// getVersionInfo 获取 PowerShell 版本信息
func (info *PSInfo) getVersionInfo(psExe string) error {
	// 获取版本号
	cmd := exec.Command(psExe, "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}
	info.Version = strings.TrimSpace(string(output))
	
	// 获取版本类型 (Desktop/Core)
	cmd = exec.Command(psExe, "-NoProfile", "-Command", "$PSVersionTable.PSEdition")
	if output, err := cmd.Output(); err == nil {
		info.Edition = strings.TrimSpace(string(output))
	}
	
	return nil
}

// getModulePaths 获取 PowerShell 模块路径
func (info *PSInfo) getModulePaths(psExe string) error {
	separator := ";"
	if runtime.GOOS != "windows" {
		separator = ":"
	}
	
	cmd := exec.Command(psExe, "-NoProfile", "-Command", 
		fmt.Sprintf("$env:PSModulePath -split '%s'", separator))
	
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	
	pathsStr := strings.TrimSpace(string(output))
	if pathsStr != "" {
		paths := strings.Split(pathsStr, "\n")
		for i, path := range paths {
			paths[i] = strings.TrimSpace(path)
		}
		info.PSModulePath = paths
	}
	
	return nil
}

// GetMajorVersion 获取 PowerShell 主版本号
func (info *PSInfo) GetMajorVersion() (int, error) {
	if info.Version == "" {
		return 0, fmt.Errorf("version not available")
	}
	
	parts := strings.Split(info.Version, ".")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid version format: %s", info.Version)
	}
	
	return strconv.Atoi(parts[0])
}

// IsCore 检查是否为 PowerShell Core
func (info *PSInfo) IsCore() bool {
	return info.Edition == "Core"
}

// IsDesktop 检查是否为 Windows PowerShell Desktop
func (info *PSInfo) IsDesktop() bool {
	return info.Edition == "Desktop"
}

// SupportsClasses 检查是否支持 PowerShell 类
func (info *PSInfo) SupportsClasses() bool {
	majorVersion, err := info.GetMajorVersion()
	if err != nil {
		return false
	}
	return majorVersion >= 5
}

// ExecuteCommand 执行 PowerShell 命令
func ExecuteCommand(command string, elevated bool) error {
	psExe, err := findPowerShellExecutable()
	if err != nil {
		return err
	}
	
	args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass"}
	
	if elevated && runtime.GOOS == "windows" {
		// 在 Windows 上使用提升权限
		elevatedScript := fmt.Sprintf(
			`Start-Process '%s' -ArgumentList '-NoProfile','-ExecutionPolicy','Bypass','-Command',@'
%s
'@ -Verb RunAs -Wait`, psExe, command)
		
		args = append(args, "-Command", elevatedScript)
	} else {
		args = append(args, "-Command", command)
	}
	
	cmd := exec.Command(psExe, args...)
	return cmd.Run()
}

// ExecuteScript 执行 PowerShell 脚本文件
func ExecuteScript(scriptPath string, elevated bool) error {
	psExe, err := findPowerShellExecutable()
	if err != nil {
		return err
	}
	
	args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath}
	
	if elevated && runtime.GOOS == "windows" {
		// 在 Windows 上使用提升权限
		elevatedArgs := strings.Join(args, `","`)
		elevatedScript := fmt.Sprintf(
			`Start-Process '%s' -ArgumentList "%s" -Verb RunAs -Wait`, 
			psExe, elevatedArgs)
		
		args = []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", elevatedScript}
	}
	
	cmd := exec.Command(psExe, args...)
	return cmd.Run()
}