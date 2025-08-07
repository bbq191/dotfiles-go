package interactive

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/sirupsen/logrus"
	"github.com/bbq191/dotfiles-go/internal/config"
	"github.com/bbq191/dotfiles-go/internal/installer"
)

// PackageSelectionScenario 包选择交互场景
type PackageSelectionScenario struct {
	// 基础属性
	name        string
	description string
	status      ScenarioStatus
	
	// 依赖注入
	installer      *installer.Installer
	packageConfig  *config.PackagesConfig
	logger         *logrus.Logger
	theme          *UITheme
	
	// 场景配置
	options        map[string]interface{}
	
	// 选择结果
	selectedPackages []string
	selectedCategories []string
	installMode     string  // "by_category", "by_package", "recommended"
}

// NewPackageSelectionScenario 创建包选择场景
func NewPackageSelectionScenario(
	installer *installer.Installer,
	packageConfig *config.PackagesConfig,
	logger *logrus.Logger,
	theme *UITheme,
) *PackageSelectionScenario {
	
	return &PackageSelectionScenario{
		name:            "package_selection",
		description:     "交互式包选择和安装",
		status:          StatusNotReady,
		installer:       installer,
		packageConfig:   packageConfig,
		logger:          logger,
		theme:           theme,
		options:         make(map[string]interface{}),
		selectedPackages: make([]string, 0),
		selectedCategories: make([]string, 0),
	}
}

// 实现 InteractiveScenario 接口
func (p *PackageSelectionScenario) Name() string {
	return p.name
}

func (p *PackageSelectionScenario) Description() string {
	return p.description
}

func (p *PackageSelectionScenario) Prerequisites() []string {
	return []string{
		"包管理器可用性检查",
		"包配置文件完整性检查",
	}
}

func (p *PackageSelectionScenario) CanExecute(ctx context.Context) (bool, error) {
	// 检查包配置是否加载
	if p.packageConfig == nil || p.packageConfig.Categories == nil {
		return false, fmt.Errorf("包配置未正确加载")
	}
	
	// 检查是否有可用的包管理器
	if len(p.packageConfig.Managers) == 0 {
		return false, fmt.Errorf("未找到可用的包管理器")
	}
	
	p.status = StatusReady
	return true, nil
}

func (p *PackageSelectionScenario) Configure(options map[string]interface{}) error {
	if options != nil {
		p.options = options
	}
	return nil
}

func (p *PackageSelectionScenario) GetStatus() ScenarioStatus {
	return p.status
}

func (p *PackageSelectionScenario) Preview() (string, error) {
	if len(p.selectedPackages) == 0 {
		return "未选择任何包", nil
	}
	
	var preview strings.Builder
	preview.WriteString(fmt.Sprintf("📦 将安装 %d 个包:\n", len(p.selectedPackages)))
	
	for _, pkg := range p.selectedPackages {
		if pkgInfo := p.findPackageInfo(pkg); pkgInfo != nil {
			preview.WriteString(fmt.Sprintf("  • %s - %s\n", pkg, pkgInfo.Description))
		} else {
			preview.WriteString(fmt.Sprintf("  • %s\n", pkg))
		}
	}
	
	return preview.String(), nil
}

func (p *PackageSelectionScenario) Execute(ctx context.Context) error {
	p.status = StatusRunning
	
	// 显示欢迎信息
	p.showWelcome()
	
	// 选择安装模式
	mode, err := p.selectInstallMode()
	if err != nil {
		p.status = StatusFailed
		return fmt.Errorf("选择安装模式失败: %w", err)
	}
	p.installMode = mode
	
	// 根据模式执行不同的选择流程
	switch mode {
	case "recommended":
		err = p.selectRecommendedPackages()
	case "by_category":
		err = p.selectByCategory()
	case "by_package":
		err = p.selectByPackage()
	case "search":
		err = p.selectBySearch()
	default:
		err = fmt.Errorf("未知的安装模式: %s", mode)
	}
	
	if err != nil {
		p.status = StatusFailed
		return err
	}
	
	// 显示选择预览并确认
	if err := p.confirmSelection(); err != nil {
		p.status = StatusCancelled
		return err
	}
	
	// 执行安装
	if err := p.executeInstallation(ctx); err != nil {
		p.status = StatusFailed
		return err
	}
	
	p.status = StatusCompleted
	return nil
}

// 内部实现方法
func (p *PackageSelectionScenario) showWelcome() {
	fmt.Printf("\n%s 智能包选择向导\n", p.theme.Icons.Package)
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("欢迎使用交互式包管理系统！\n")
	fmt.Printf("我们将引导您选择和安装适合的软件包。\n\n")
	
	// 显示包统计信息
	totalPackages := p.getTotalPackageCount()
	categoryCount := len(p.packageConfig.Categories)
	
	fmt.Printf("📊 可用资源:\n")
	fmt.Printf("  • 软件分类: %d 个\n", categoryCount)
	fmt.Printf("  • 软件包: %d 个\n", totalPackages)
	fmt.Printf("  • 包管理器: %d 个\n\n", len(p.packageConfig.Managers))
}

func (p *PackageSelectionScenario) selectInstallMode() (string, error) {
	prompt := &survey.Select{
		Message: "请选择安装方式:",
		Options: []string{
			"推荐配置 - 自动选择常用软件包",
			"按分类选择 - 浏览软件分类",
			"逐个选择 - 查看所有软件包",
			"搜索模式 - 按名称或标签搜索",
		},
		Help:    "选择最适合您的安装方式",
	}
	
	var selection string
	if err := survey.AskOne(prompt, &selection); err != nil {
		return "", err
	}
	
	switch {
	case strings.HasPrefix(selection, "推荐配置"):
		return "recommended", nil
	case strings.HasPrefix(selection, "按分类选择"):
		return "by_category", nil
	case strings.HasPrefix(selection, "逐个选择"):
		return "by_package", nil
	case strings.HasPrefix(selection, "搜索模式"):
		return "search", nil
	default:
		return "recommended", nil
	}
}

func (p *PackageSelectionScenario) selectRecommendedPackages() error {
	// 获取推荐包
	recommended := p.getRecommendedPackages()
	
	if len(recommended) == 0 {
		fmt.Printf("%s 未找到推荐包，切换到分类选择模式\n", p.theme.Icons.Warning)
		return p.selectByCategory()
	}
	
	// 显示推荐包信息
	fmt.Printf("\n%s 推荐软件包 (%d 个):\n", p.theme.Icons.Info, len(recommended))
	for _, pkg := range recommended {
		if pkgInfo := p.findPackageInfo(pkg); pkgInfo != nil {
			fmt.Printf("  • %s - %s\n", pkg, pkgInfo.Description)
		}
	}
	
	// 询问是否接受推荐
	var accept bool
	prompt := &survey.Confirm{
		Message: "是否安装所有推荐的软件包?",
		Default: true,
		Help:    "这些包是基于您的平台和常用需求推荐的",
	}
	
	if err := survey.AskOne(prompt, &accept); err != nil {
		return err
	}
	
	if accept {
		p.selectedPackages = recommended
		return nil
	}
	
	// 用户拒绝推荐，切换到自定义选择
	fmt.Printf("\n%s 切换到自定义选择模式...\n", p.theme.Icons.Configure)
	return p.selectByCategory()
}

func (p *PackageSelectionScenario) selectByCategory() error {
	// 获取排序后的分类列表
	categories := p.getSortedCategories()
	
	// 创建分类选择选项
	var categoryOptions []string
	for _, cat := range categories {
		categoryInfo := p.packageConfig.Categories[cat]
		packageCount := len(categoryInfo.Packages)
		option := fmt.Sprintf("%s (%d 个包) - %s", 
			cat, packageCount, categoryInfo.Description)
		categoryOptions = append(categoryOptions, option)
	}
	
	// 多选分类
	prompt := &survey.MultiSelect{
		Message: "选择要安装的软件分类 (空格键选择，回车键确认):",
		Options: categoryOptions,
		Help:    "可以选择多个分类，稍后可以在分类内进一步选择具体软件包",
	}
	
	var selectedOptions []string
	if err := survey.AskOne(prompt, &selectedOptions); err != nil {
		return err
	}
	
	if len(selectedOptions) == 0 {
		return fmt.Errorf("未选择任何分类")
	}
	
	// 提取分类名称
	for _, option := range selectedOptions {
		parts := strings.Split(option, " ")
		if len(parts) > 0 {
			p.selectedCategories = append(p.selectedCategories, parts[0])
		}
	}
	
	// 为每个选择的分类选择具体包
	for _, category := range p.selectedCategories {
		if err := p.selectPackagesInCategory(category); err != nil {
			return err
		}
	}
	
	return nil
}

func (p *PackageSelectionScenario) selectPackagesInCategory(category string) error {
	categoryInfo, exists := p.packageConfig.Categories[category]
	if !exists {
		return fmt.Errorf("分类 %s 不存在", category)
	}
	
	fmt.Printf("\n%s 分类: %s\n", p.theme.Icons.Category, category)
	fmt.Printf("描述: %s\n\n", categoryInfo.Description)
	
	// 创建包选择选项
	var packageOptions []string
	var packageNames []string
	
	for name, pkg := range categoryInfo.Packages {
		option := fmt.Sprintf("%s - %s", name, pkg.Description)
		if pkg.Optional {
			option += " [可选]"
		}
		packageOptions = append(packageOptions, option)
		packageNames = append(packageNames, name)
	}
	
	// 默认选择必需包
	var defaultSelection []string
	for i, name := range packageNames {
		if pkg, exists := categoryInfo.Packages[name]; exists && !pkg.Optional {
			defaultSelection = append(defaultSelection, packageOptions[i])
		}
	}
	
	// 多选包
	prompt := &survey.MultiSelect{
		Message: fmt.Sprintf("选择 %s 分类中的软件包:", category),
		Options: packageOptions,
		Default: defaultSelection,
		Help:    "空格键选择/取消，上下键导航，回车键确认",
	}
	
	var selectedOptions []string
	if err := survey.AskOne(prompt, &selectedOptions); err != nil {
		return err
	}
	
	// 提取包名称并添加到选择列表
	for _, option := range selectedOptions {
		parts := strings.Split(option, " - ")
		if len(parts) > 0 {
			packageName := parts[0]
			// 避免重复添加
			found := false
			for _, existing := range p.selectedPackages {
				if existing == packageName {
					found = true
					break
				}
			}
			if !found {
				p.selectedPackages = append(p.selectedPackages, packageName)
			}
		}
	}
	
	return nil
}

func (p *PackageSelectionScenario) selectByPackage() error {
	// 获取所有包的列表
	allPackages := p.getAllPackages()
	
	// 创建包选择选项
	var packageOptions []string
	for _, pkg := range allPackages {
		option := fmt.Sprintf("%s - %s", pkg.Name, pkg.Description)
		if pkg.Optional {
			option += " [可选]"
		}
		packageOptions = append(packageOptions, option)
	}
	
	// 多选包
	prompt := &survey.MultiSelect{
		Message: "选择要安装的软件包 (空格键选择，回车键确认):",
		Options: packageOptions,
		Help:    fmt.Sprintf("共 %d 个软件包可选择", len(packageOptions)),
	}
	
	var selectedOptions []string
	if err := survey.AskOne(prompt, &selectedOptions); err != nil {
		return err
	}
	
	if len(selectedOptions) == 0 {
		return fmt.Errorf("未选择任何软件包")
	}
	
	// 提取包名称
	for _, option := range selectedOptions {
		parts := strings.Split(option, " - ")
		if len(parts) > 0 {
			p.selectedPackages = append(p.selectedPackages, parts[0])
		}
	}
	
	return nil
}

func (p *PackageSelectionScenario) selectBySearch() error {
	for {
		// 搜索关键词输入
		var keyword string
		prompt := &survey.Input{
			Message: "输入搜索关键词 (包名或标签):",
			Help:    "可以搜索包名、标签或描述中的关键词",
		}
		
		if err := survey.AskOne(prompt, &keyword); err != nil {
			return err
		}
		
		if keyword == "" {
			break
		}
		
		// 执行搜索
		results := p.searchPackages(keyword)
		if len(results) == 0 {
			fmt.Printf("%s 未找到匹配的软件包，请尝试其他关键词\n", p.theme.Icons.Warning)
			continue
		}
		
		// 显示搜索结果并选择
		fmt.Printf("\n%s 找到 %d 个匹配的软件包:\n", p.theme.Icons.Search, len(results))
		
		var resultOptions []string
		for _, pkg := range results {
			option := fmt.Sprintf("%s - %s", pkg.Name, pkg.Description)
			resultOptions = append(resultOptions, option)
		}
		
		// 多选搜索结果
		selectPrompt := &survey.MultiSelect{
			Message: "从搜索结果中选择软件包:",
			Options: resultOptions,
		}
		
		var selectedOptions []string
		if err := survey.AskOne(selectPrompt, &selectedOptions); err != nil {
			return err
		}
		
		// 添加选择的包
		for _, option := range selectedOptions {
			parts := strings.Split(option, " - ")
			if len(parts) > 0 {
				packageName := parts[0]
				// 避免重复添加
				found := false
				for _, existing := range p.selectedPackages {
					if existing == packageName {
						found = true
						break
					}
				}
				if !found {
					p.selectedPackages = append(p.selectedPackages, packageName)
				}
			}
		}
		
		// 询问是否继续搜索
		var continueSearch bool
		continuePrompt := &survey.Confirm{
			Message: "是否继续搜索其他软件包?",
			Default: false,
		}
		
		if err := survey.AskOne(continuePrompt, &continueSearch); err != nil {
			return err
		}
		
		if !continueSearch {
			break
		}
	}
	
	if len(p.selectedPackages) == 0 {
		return fmt.Errorf("未选择任何软件包")
	}
	
	return nil
}

func (p *PackageSelectionScenario) confirmSelection() error {
	if len(p.selectedPackages) == 0 {
		return fmt.Errorf("未选择任何软件包")
	}
	
	// 显示选择预览
	fmt.Printf("\n%s 安装预览\n", p.theme.Icons.Preview)
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("即将安装以下 %d 个软件包:\n\n", len(p.selectedPackages))
	
	for i, pkg := range p.selectedPackages {
		if pkgInfo := p.findPackageInfo(pkg); pkgInfo != nil {
			fmt.Printf("%2d. %s\n    %s\n", i+1, pkg, pkgInfo.Description)
			if len(pkgInfo.Tags) > 0 {
				fmt.Printf("    标签: %s\n", strings.Join(pkgInfo.Tags, ", "))
			}
		} else {
			fmt.Printf("%2d. %s\n", i+1, pkg)
		}
		fmt.Println()
	}
	
	// 询问确认
	var confirm bool
	prompt := &survey.Confirm{
		Message: "确认安装这些软件包吗?",
		Default: true,
		Help:    "选择 Yes 开始安装，选择 No 取消操作",
	}
	
	if err := survey.AskOne(prompt, &confirm); err != nil {
		return err
	}
	
	if !confirm {
		return fmt.Errorf("用户取消了安装操作")
	}
	
	return nil
}

func (p *PackageSelectionScenario) executeInstallation(ctx context.Context) error {
	fmt.Printf("\n%s 开始安装软件包...\n", p.theme.Icons.Install)
	
	// 使用现有的安装器执行安装
	options := installer.InstallOptions{
		Force:      false,
		Parallel:   false, // 交互模式使用串行安装更安全
		MaxWorkers: 1,
		Quiet:      false,
		DryRun:     false,
		Verbose:    true,
	}
	
	results, err := p.installer.InstallPackages(ctx, p.selectedPackages, options)
	if err != nil {
		return err
	}
	
	// 检查安装结果
	failed := 0
	for _, result := range results {
		if !result.Success && !result.Skipped {
			failed++
		}
	}
	
	if failed > 0 {
		return fmt.Errorf("有 %d 个包安装失败", failed)
	}
	
	return nil
}

// 辅助方法
func (p *PackageSelectionScenario) getTotalPackageCount() int {
	count := 0
	for _, category := range p.packageConfig.Categories {
		count += len(category.Packages)
	}
	return count
}

func (p *PackageSelectionScenario) getSortedCategories() []string {
	var categories []string
	for name := range p.packageConfig.Categories {
		categories = append(categories, name)
	}
	
	// 按优先级排序
	sort.Slice(categories, func(i, j int) bool {
		cat1 := p.packageConfig.Categories[categories[i]]
		cat2 := p.packageConfig.Categories[categories[j]]
		return cat1.Priority < cat2.Priority
	})
	
	return categories
}

func (p *PackageSelectionScenario) getRecommendedPackages() []string {
	var recommended []string
	
	// 遍历所有分类，收集推荐包
	for _, category := range p.packageConfig.Categories {
		for name, pkg := range category.Packages {
			// 推荐条件：不是可选包 且 优先级高的分类
			if !pkg.Optional && category.Priority <= 3 {
				recommended = append(recommended, name)
			}
		}
	}
	
	return recommended
}

func (p *PackageSelectionScenario) findPackageInfo(packageName string) *config.PackageInfo {
	for _, category := range p.packageConfig.Categories {
		if pkg, exists := category.Packages[packageName]; exists {
			return &pkg
		}
	}
	return nil
}

// PackageSearchResult 搜索结果
type PackageSearchResult struct {
	Name        string
	Description string
	Category    string
	Tags        []string
	Optional    bool
}

func (p *PackageSelectionScenario) getAllPackages() []PackageSearchResult {
	var packages []PackageSearchResult
	
	for categoryName, category := range p.packageConfig.Categories {
		for name, pkg := range category.Packages {
			packages = append(packages, PackageSearchResult{
				Name:        name,
				Description: pkg.Description,
				Category:    categoryName,
				Tags:        pkg.Tags,
				Optional:    pkg.Optional,
			})
		}
	}
	
	return packages
}

func (p *PackageSelectionScenario) searchPackages(keyword string) []PackageSearchResult {
	var results []PackageSearchResult
	keyword = strings.ToLower(keyword)
	
	allPackages := p.getAllPackages()
	
	for _, pkg := range allPackages {
		// 搜索包名
		if strings.Contains(strings.ToLower(pkg.Name), keyword) {
			results = append(results, pkg)
			continue
		}
		
		// 搜索描述
		if strings.Contains(strings.ToLower(pkg.Description), keyword) {
			results = append(results, pkg)
			continue
		}
		
		// 搜索标签
		for _, tag := range pkg.Tags {
			if strings.Contains(strings.ToLower(tag), keyword) {
				results = append(results, pkg)
				break
			}
		}
	}
	
	return results
}