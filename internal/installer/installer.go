package installer

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// InstallPackage 安装单个包 - MVP核心功能
func (i *Installer) InstallPackage(ctx context.Context, packageName string, opts InstallOptions) (*InstallResult, error) {
	startTime := time.Now()
	
	result := &InstallResult{
		PackageName: packageName,
		Success:     false,
	}
	
	// 选择包管理器
	manager := i.SelectManager()
	if manager == nil {
		err := fmt.Errorf("没有找到可用的包管理器")
		i.logger.Error(err)
		result.Error = err
		return result, err
	}
	
	result.Manager = manager.Name()
	i.logger.Infof("选择包管理器: %s 安装包: %s", manager.Name(), packageName)
	
	// 检查是否需要跳过已安装的包
	if !opts.Force && manager.IsInstalled(packageName) {
		i.logger.Infof("包 %s 已安装，跳过安装", packageName)
		result.Success = true
		result.Skipped = true
		result.Duration = time.Since(startTime).Seconds()
		return result, nil
	}
	
	// 执行安装
	if opts.DryRun {
		i.logger.Infof("[DRY RUN] 将使用 %s 安装 %s", manager.Name(), packageName)
		result.Success = true
		result.Duration = time.Since(startTime).Seconds()
		return result, nil
	}
	
	// 实际安装
	err := manager.Install(ctx, packageName)
	result.Duration = time.Since(startTime).Seconds()
	
	if err != nil {
		i.logger.Errorf("安装包 %s 失败: %v", packageName, err)
		result.Error = err
		return result, err
	}
	
	result.Success = true
	i.logger.Infof("成功安装包 %s，耗时: %.2f秒", packageName, result.Duration)
	
	return result, nil
}

// InstallPackages 安装多个包 - 支持进度显示
func (i *Installer) InstallPackages(ctx context.Context, packages []string, opts InstallOptions) ([]*InstallResult, error) {
	results := make([]*InstallResult, 0, len(packages))
	
	// 创建进度管理器
	progressMgr := NewProgressManager(packages, i.logger, opts.Quiet)
	
	// 启动进度显示（除非是quiet模式）
	if !opts.Quiet {
		progressMgr.Start()
		defer progressMgr.Close()
	}
	
	i.logger.Infof("开始批量安装 %d 个包", len(packages))
	
	for _, pkg := range packages {
		select {
		case <-ctx.Done():
			i.logger.Warn("安装被取消")
			return results, ctx.Err()
		default:
			// 发送开始安装事件
			progressMgr.SendEvent(ProgressEvent{
				Type:        ProgressStart,
				PackageName: pkg,
				Message:     "开始安装",
			})
			
			result, err := i.InstallPackage(ctx, pkg, opts)
			results = append(results, result)
			
			// 添加结果到进度管理器
			progressMgr.AddResult(result)
			
			// 发送相应的进度事件
			if err != nil {
				progressMgr.SendEvent(ProgressEvent{
					Type:        ProgressFail,
					PackageName: pkg,
					Manager:     result.Manager,
					Error:       err,
				})
				
				if !opts.Force {
					i.logger.Errorf("安装包 %s 失败，停止批量安装", pkg)
					break
				}
			} else if result.Success {
				if result.Skipped {
					progressMgr.SendEvent(ProgressEvent{
						Type:        ProgressSkip,
						PackageName: pkg,
						Manager:     result.Manager,
						Message:     "包已存在",
					})
				} else {
					progressMgr.SendEvent(ProgressEvent{
						Type:        ProgressSuccess,
						PackageName: pkg,
						Manager:     result.Manager,
						Message:     "安装成功",
					})
				}
			}
		}
	}
	
	// 显示总结（除非是quiet模式）
	if !opts.Quiet {
		// 等待进度显示完成
		time.Sleep(100 * time.Millisecond)
		progressMgr.PrintSummaryTable()
	}
	
	// 统计结果
	successful := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
	}
	
	i.logger.Infof("批量安装完成 - 成功: %d, 失败: %d", successful, failed)
	
	return results, nil
}

// InitializeManagers 初始化并注册所有包管理器
func (i *Installer) InitializeManagers() {
	i.logger.Info("初始化包管理器")
	
	// 注册 Yay (Arch Linux + AUR) - 优先级最高的AUR管理器
	yay := NewYayManager(i.logger)
	i.RegisterManager(yay)
	
	// 注册 Pacman (Linux) - 官方包管理器
	pacman := NewPacmanManager(i.logger)
	i.RegisterManager(pacman)
	
	// 注册 Winget (Windows)
	winget := NewWingetManager(i.logger)
	i.RegisterManager(winget)
	
	// 排序管理器（按优先级）
	sort.Slice(i.managers, func(a, b int) bool {
		return i.managers[a].Priority() < i.managers[b].Priority()
	})
	
	// 输出可用管理器信息
	available := i.GetAvailableManagers()
	if len(available) == 0 {
		i.logger.Warn("没有找到可用的包管理器")
	} else {
		i.logger.Infof("发现 %d 个可用的包管理器:", len(available))
		for _, manager := range available {
			i.logger.Infof("  - %s (优先级: %d)", manager.Name(), manager.Priority())
		}
	}
}