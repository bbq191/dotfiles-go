package installer

import (
	"context"
	"os/exec"
	"strings"
	"runtime"
	
	"github.com/sirupsen/logrus"
)

// WingetManager Winget包管理器实现
type WingetManager struct {
	logger *logrus.Logger
}

// NewWingetManager 创建Winget管理器实例
func NewWingetManager(logger *logrus.Logger) *WingetManager {
	return &WingetManager{
		logger: logger,
	}
}

// Name 返回包管理器名称
func (w *WingetManager) Name() string {
	return "winget"
}

// IsAvailable 检查winget是否可用
func (w *WingetManager) IsAvailable() bool {
	// Winget 只在 Windows 上可用
	if runtime.GOOS != "windows" {
		w.logger.Debug("Winget 不适用于非Windows系统")
		return false
	}
	
	_, err := exec.LookPath("winget")
	available := err == nil
	w.logger.Debugf("Winget 可用性检查: %v", available)
	return available
}

// Install 安装包
func (w *WingetManager) Install(ctx context.Context, packageName string) error {
	w.logger.Infof("使用 Winget 安装包: %s", packageName)
	
	// 检查是否已安装（winget暂不支持准确的已安装检查，直接尝试安装）
	
	// 构建安装命令
	args := []string{"install", "--id", packageName, "--silent", "--accept-package-agreements", "--accept-source-agreements"}
	cmd := exec.CommandContext(ctx, "winget", args...)
	
	w.logger.Debugf("执行命令: winget %s", strings.Join(args, " "))
	
	// 设置命令输出
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		// Winget 有时候即使安装成功也会返回非零退出码，需要检查输出内容
		outputStr := string(output)
		if strings.Contains(outputStr, "Successfully installed") || 
		   strings.Contains(outputStr, "already installed") {
			w.logger.Infof("包 %s 安装成功或已存在", packageName)
			return nil
		}
		
		w.logger.Errorf("安装 %s 失败: %v", packageName, err)
		w.logger.Debugf("命令输出: %s", outputStr)
		return err
	}
	
	w.logger.Infof("成功安装 %s", packageName)
	w.logger.Debugf("安装输出: %s", string(output))
	
	return nil
}

// IsInstalled 检查包是否已安装
func (w *WingetManager) IsInstalled(packageName string) bool {
	// Winget的包状态检查相对复杂，这里简化实现
	cmd := exec.Command("winget", "list", "--id", packageName)
	err := cmd.Run()
	
	installed := err == nil
	w.logger.Debugf("包 %s 安装状态检查 (简化): %v", packageName, installed)
	
	return installed
}

// Priority 返回优先级
func (w *WingetManager) Priority() int {
	return 2 // Winget 优先级稍低于系统原生包管理器
}

// Search 搜索包（额外功能）
func (w *WingetManager) Search(query string) ([]string, error) {
	cmd := exec.Command("winget", "search", query)
	output, err := cmd.Output()
	
	if err != nil {
		return nil, err
	}
	
	// 解析搜索结果
	results := make([]string, 0)
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Name") && !strings.HasPrefix(line, "---") {
			// 简化解析，提取第一个字段作为包ID
			fields := strings.Fields(line)
			if len(fields) > 0 {
				results = append(results, fields[0])
			}
		}
	}
	
	return results, nil
}