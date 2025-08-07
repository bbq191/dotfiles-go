package installer

import (
	"context"
	"os/exec"
	"strings"
	
	"github.com/sirupsen/logrus"
)

// PacmanManager Pacman包管理器实现
type PacmanManager struct {
	logger *logrus.Logger
}

// NewPacmanManager 创建Pacman管理器实例
func NewPacmanManager(logger *logrus.Logger) *PacmanManager {
	return &PacmanManager{
		logger: logger,
	}
}

// Name 返回包管理器名称
func (p *PacmanManager) Name() string {
	return "pacman"
}

// IsAvailable 检查pacman是否可用
func (p *PacmanManager) IsAvailable() bool {
	_, err := exec.LookPath("pacman")
	available := err == nil
	p.logger.Debugf("Pacman 可用性检查: %v", available)
	return available
}

// Install 安装包
func (p *PacmanManager) Install(ctx context.Context, packageName string) error {
	p.logger.Infof("使用 Pacman 安装包: %s", packageName)
	
	// 检查是否已安装
	if p.IsInstalled(packageName) {
		p.logger.Infof("包 %s 已安装，跳过", packageName)
		return nil
	}
	
	// 构建安装命令
	args := []string{"-S", "--noconfirm", packageName}
	cmd := exec.CommandContext(ctx, "sudo", append([]string{"pacman"}, args...)...)
	
	p.logger.Debugf("执行命令: sudo pacman %s", strings.Join(args, " "))
	
	// 设置命令输出
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		p.logger.Errorf("安装 %s 失败: %v", packageName, err)
		p.logger.Debugf("命令输出: %s", string(output))
		return err
	}
	
	p.logger.Infof("成功安装 %s", packageName)
	p.logger.Debugf("安装输出: %s", string(output))
	
	return nil
}

// IsInstalled 检查包是否已安装
func (p *PacmanManager) IsInstalled(packageName string) bool {
	cmd := exec.Command("pacman", "-Q", packageName)
	err := cmd.Run()
	
	installed := err == nil
	p.logger.Debugf("包 %s 安装状态: %v", packageName, installed)
	
	return installed
}

// Priority 返回优先级
func (p *PacmanManager) Priority() int {
	return 1 // Pacman 为官方包管理器，优先级较高
}

// GetPackageInfo 获取包信息（额外功能）
func (p *PacmanManager) GetPackageInfo(packageName string) (map[string]string, error) {
	cmd := exec.Command("pacman", "-Si", packageName)
	output, err := cmd.Output()
	
	if err != nil {
		return nil, err
	}
	
	// 解析包信息
	info := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}
	}
	
	return info, nil
}