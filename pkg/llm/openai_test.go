package llm

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/turtacn/kubestack-ai/pkg/llm/chat"
	"k8s.io/klog/v2"
)

func init() {
	os.Setenv("OPENAI_API_KEY", "xxx")
	os.Setenv("OPENAI_API_BASE", "url")
	os.Setenv("OPENAI_MODEL", "qwen3-32b")
	Init()
}

func TestOpenaiClient(t *testing.T) {
	client, err := NewClient(context.TODO(), "openai", WithHeaders(map[string]string{"Authorization": "xxx"}))
	if err != nil {
		t.Fatal(err)
	}
	chatSession := client.StartChat("我正在对大模型api进行测试，你只需要简单回复即可", "qwen3-32b")
	iter, err := chatSession.SendStreaming(context.TODO(), &chat.Message{
		Source:  chat.MessageSourceUser,
		Payload: "你是谁",
	})

	if err != nil {
		t.Fatal(err)
	}
	for response, err := range iter {
		if err != nil {
			klog.Errorf("接收流式响应失败: %v", err)
			return
		}

		// 获取候选响应
		candidates := response.Candidates()
		if len(candidates) == 0 {
			continue
		}

		// 获取第一个候选响应
		candidate := candidates[0]
		parts := candidate.Parts()

		for _, part := range parts {
			// 获取文本内容
			if text, ok := part.AsText(); ok {
				fmt.Print(text)
			}
		}
	}
	fmt.Println() // 换行
}

// TestOpenaiSessionContext 测试一个session中多轮对话是否能够保留上下文
func TestOpenaiSessionContext(t *testing.T) {
	client, err := NewClient(context.TODO(), "openai", WithHeaders(map[string]string{"Authorization": "sk-KskGcDMEQWGncNHr6bE2Ee61F22b40F8A1C09c8b150968Ff"}))
	if err != nil {
		t.Fatal(err)
	}

	// 创建聊天session，设置系统提示
	chatSession := client.StartChat("我是一个测试助手，请记住我们的对话内容", "qwen3-32b")

	// 第一轮对话：介绍自己
	fmt.Println("=== 第一轮对话 ===")
	iter1, err := chatSession.SendStreaming(context.TODO(), &chat.Message{
		Source:  chat.MessageSourceUser,
		Payload: "你好，我叫张三，今年25岁，是一名软件工程师",
	})

	if err != nil {
		t.Fatal(err)
	}

	var response1 string
	for response, err := range iter1 {
		if err != nil {
			klog.Errorf("接收第一轮流式响应失败: %v", err)
			return
		}

		candidates := response.Candidates()
		if len(candidates) == 0 {
			continue
		}

		candidate := candidates[0]
		parts := candidate.Parts()

		for _, part := range parts {
			if text, ok := part.AsText(); ok {
				fmt.Print(text)
				response1 += text
			}
		}
	}
	fmt.Println()

	// 第二轮对话：询问个人信息，测试是否记得第一轮的信息
	fmt.Println("\n=== 第二轮对话 ===")
	iter2, err := chatSession.SendStreaming(context.TODO(), &chat.Message{
		Source:  chat.MessageSource(chat.MessageSourceUser),
		Payload: "请告诉我我的名字、年龄和职业是什么？",
	})

	if err != nil {
		t.Fatal(err)
	}

	var response2 string
	for response, err := range iter2 {
		if err != nil {
			klog.Errorf("接收第二轮流式响应失败: %v", err)
			return
		}

		candidates := response.Candidates()
		if len(candidates) == 0 {
			continue
		}

		candidate := candidates[0]
		parts := candidate.Parts()

		for _, part := range parts {
			if text, ok := part.AsText(); ok {
				fmt.Print(text)
				response2 += text
			}
		}
	}
	fmt.Println()

	// 第三轮对话：进一步测试上下文理解
	fmt.Println("\n=== 第三轮对话 ===")
	iter3, err := chatSession.SendStreaming(context.TODO(), &chat.Message{
		Source:  chat.MessageSourceUser,
		Payload: "根据我们之前的对话，你觉得我的工作主要涉及什么技术领域？",
	})

	if err != nil {
		t.Fatal(err)
	}

	var response3 string
	for response, err := range iter3 {
		if err != nil {
			klog.Errorf("接收第三轮流式响应失败: %v", err)
			return
		}

		candidates := response.Candidates()
		if len(candidates) == 0 {
			continue
		}

		candidate := candidates[0]
		parts := candidate.Parts()

		for _, part := range parts {
			if text, ok := part.AsText(); ok {
				fmt.Print(text)
				response3 += text
			}
		}
	}
	fmt.Println()

	// 验证上下文保留情况
	fmt.Println("\n=== 上下文保留验证 ===")

	// 检查第二轮回复是否包含第一轮的个人信息
	contextPreserved := true
	if !containsKeywords(response2, []string{"张三", "25", "软件工程师"}) {
		t.Errorf("第二轮回复未正确记住用户个人信息: %s", response2)
		contextPreserved = false
	}

	// 检查第三轮回复是否显示对软件工程领域的理解
	if !containsKeywords(response3, []string{"软件", "程序", "开发", "编程", "技术"}) {
		t.Errorf("第三轮回复未显示对软件工程领域的理解: %s", response3)
		contextPreserved = false
	}

	if contextPreserved {
		fmt.Println("✅ 上下文保留测试通过：多轮对话成功保留了对话历史")
	} else {
		fmt.Println("❌ 上下文保留测试失败：未正确保留对话历史")
	}
}

// TestOpenaiFunctionCall 测试工具调用功能
func TestOpenaiFunctionCall(t *testing.T) {
	client, err := NewClient(context.TODO(), "openai", WithHeaders(map[string]string{"Authorization": "sk-KskGcDMEQWGncNHr6bE2Ee61F22b40F8A1C09c8b150968Ff"}))
	if err != nil {
		t.Fatal(err)
	}

	// 创建聊天session
	chatSession := client.StartChat("你是一个智能助手，可以使用提供的工具来帮助用户", "qwen3-32b")

	// 定义工具函数
	functionDefinitions := []*chat.FunctionDefinition{
		{
			Name:        "get_weather",
			Description: "获取指定城市的天气信息",
			Parameters: &chat.Schema{
				Type: "object",
				Properties: map[string]*chat.Schema{
					"city": {
						Type:        "string",
						Description: "城市名称",
					},
					"date": {
						Type:        "string",
						Description: "日期，格式为 YYYY-MM-DD",
					},
				},
				Required: []string{"city"},
			},
		},
		{
			Name:        "calculate",
			Description: "执行数学计算",
			Parameters: &chat.Schema{
				Type: "object",
				Properties: map[string]*chat.Schema{
					"expression": {
						Type:        "string",
						Description: "数学表达式，例如 '2 + 3 * 4'",
					},
				},
				Required: []string{"expression"},
			},
		},
	}

	// 设置工具定义
	err = chatSession.SetFunctionDefinitions(functionDefinitions)
	if err != nil {
		t.Fatalf("设置工具定义失败: %v", err)
	}

	fmt.Println("=== 工具调用测试 ===")

	// 发送需要工具调用的消息
	iter, err := chatSession.SendStreaming(context.TODO(), &chat.Message{
		Source:  chat.MessageSourceUser,
		Payload: "请帮我计算 15 * 8 + 3 的结果，并查询北京今天的天气",
	})

	if err != nil {
		t.Fatal(err)
	}

	var functionCalls []chat.FunctionCall
	var responseText string

	for response, err := range iter {
		if err != nil {
			klog.Errorf("接收流式响应失败: %v", err)
			return
		}

		candidates := response.Candidates()
		if len(candidates) == 0 {
			continue
		}

		candidate := candidates[0]
		parts := candidate.Parts()

		for _, part := range parts {
			// 获取文本内容
			if text, ok := part.AsText(); ok {
				fmt.Print(text)
				responseText += text
			}

			// 获取函数调用
			if calls, ok := part.AsFunctionCalls(); ok {
				functionCalls = append(functionCalls, calls...)
				for _, call := range calls {
					fmt.Printf("\n🔧 工具调用: %s(%v)\n", call.Name, call.Arguments)
				}
			}
		}
	}
	fmt.Println()

	// 验证工具调用
	fmt.Println("\n=== 工具调用验证 ===")

	if len(functionCalls) == 0 {
		t.Error("❌ 测试失败：未检测到工具调用")
		fmt.Println("❌ 工具调用测试失败：模型未调用任何工具")
		return
	}

	// 检查是否调用了预期的工具
	expectedTools := map[string]bool{
		"calculate":   false,
		"get_weather": false,
	}

	for _, call := range functionCalls {
		if _, exists := expectedTools[call.Name]; exists {
			expectedTools[call.Name] = true
			fmt.Printf("✅ 成功调用工具: %s\n", call.Name)

			// 验证参数
			if call.Name == "calculate" {
				if expr, ok := call.Arguments["expression"].(string); ok && expr != "" {
					fmt.Printf("✅ calculate 工具参数正确: %s\n", expr)
				} else {
					t.Errorf("❌ calculate 工具缺少 expression 参数")
				}
			}

			if call.Name == "get_weather" {
				if city, ok := call.Arguments["city"].(string); ok && city != "" {
					fmt.Printf("✅ get_weather 工具参数正确: %s\n", city)
				} else {
					t.Errorf("❌ get_weather 工具缺少 city 参数")
				}
			}
		}
	}

	// 检查是否调用了所有预期的工具
	allToolsCalled := true
	for tool, called := range expectedTools {
		if !called {
			t.Errorf("❌ 未调用预期工具: %s", tool)
			allToolsCalled = false
		}
	}

	if allToolsCalled && len(functionCalls) > 0 {
		fmt.Println("✅ 工具调用测试通过：成功调用了预期的工具函数")
	} else {
		fmt.Println("❌ 工具调用测试失败：未正确调用所有预期工具")
	}
}

// containsKeywords 检查文本中是否包含指定的关键词
func containsKeywords(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if contains(text, keyword) {
			return true
		}
	}
	return false
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

// findSubstring 在字符串中查找子字符串
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
