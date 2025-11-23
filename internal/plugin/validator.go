package plugin

import (
	"fmt"
	"log"
	"strings"
)

// Validator 插件验证器
type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// Validate 验证插件是否实现了所有必要方法
func (v *Validator) Validate(plugin interface{}) bool {
	// Step 1: 检查是否实现DiagnosticPlugin接口
	_, ok := plugin.(DiagnosticPlugin)
	if !ok {
		log.Println("插件未实现DiagnosticPlugin接口")
		return false
	}

	// Step 2: 验证插件元数据
	p := plugin.(DiagnosticPlugin)
	if p.Name() == "" {
		log.Println("插件名称不能为空")
		return false
	}

	if len(p.SupportedTypes()) == 0 {
		log.Println("插件必须声明至少一个支持的中间件类型")
		return false
	}

	if p.Version() == "" {
		log.Println("插件版本号不能为空")
		return false
	}

	// Step 3: 验证插件安全性（可选）
	// 例如：检查插件是否尝试访问敏感资源
	if err := v.checkSecurity(p); err != nil {
		log.Printf("插件安全检查失败: %v", err)
		return false
	}

	return true
}

// checkSecurity 安全检查（防止恶意插件）
func (v *Validator) checkSecurity(plugin DiagnosticPlugin) error {
	// 1. 检查插件是否尝试访问文件系统的敏感路径
	// 2. 检查是否尝试建立不安全的网络连接
	// 3. 检查是否尝试执行系统命令

	// 示例：简单的命名规范检查
	name := plugin.Name()
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		return fmt.Errorf("插件名称包含非法字符")
	}

	return nil
}
