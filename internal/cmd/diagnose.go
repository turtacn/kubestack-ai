package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/turtacn/kubestack-ai/internal/ai"
	"github.com/turtacn/kubestack-ai/internal/collectors"
	"github.com/turtacn/kubestack-ai/internal/diagnosis"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// NewDiagnoseCmd 创建诊断命令。NewDiagnoseCmd creates diagnose command.
func NewDiagnoseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnose [middleware]",
		Short: "诊断中间件问题。Diagnose middleware issues",
		Long: `对指定的中间件进行全面诊断，包括性能、配置和日志分析，并提供AI驱动的优化建议。
Perform comprehensive diagnosis of specified middleware including performance, configuration, and log analysis with AI-driven recommendations.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			middleware := args[0]
			environment, _ := cmd.Flags().GetString("environment")
			namespace, _ := cmd.Flags().GetString("namespace")
			labelSelector, _ := cmd.Flags().GetString("labels")
			query, _ := cmd.Flags().GetString("query")

			logging.Logger.Infof("Starting diagnosis for %s in %s environment", middleware, environment)

			// 初始化收集器。Initialize collector.
			var collector collectors.Collector
			if environment == "kubernetes" {
				// 初始化Kubernetes客户端和收集器。Initialize Kubernetes client and collector.
				config, err := rest.InClusterConfig()
				if err != nil {
					config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
					if err != nil {
						logging.Logger.Fatalf("Failed to create Kubernetes config: %v", err)
						os.Exit(1)
					}
				}

				clientset, err := kubernetes.NewForConfig(config)
				if err != nil {
					logging.Logger.Fatalf("Failed to create Kubernetes client: %v", err)
					os.Exit(1)
				}

				collector = collectors.NewK8sCollector(clientset, namespace, labelSelector)
			} else {
				// 使用裸机收集器。Use bare metal collector.
				collector = collectors.NewBareCollector(middleware)
			}

			// 初始化LLM和RAG。Initialize LLM and RAG.
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				logging.Logger.Fatal("OPENAI_API_KEY environment variable not set")
				os.Exit(1)
			}

			llm := ai.NewLLM(apiKey, "gpt-4")
			rag := ai.NewRAG()

			// 初始化诊断引擎。Initialize diagnosis engine.
			engine := diagnosis.NewEngine(collector, plugins.Manager, llm, rag)

			// 执行诊断。Perform diagnosis.
			var result *models.DiagnosisResult
			var err error
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if query != "" {
				result, err = engine.DiagnoseWithQuery(ctx, middleware, query)
			} else {
				params := map[string]string{
					"namespace":      namespace,
					"label_selector": labelSelector,
				}
				result, err = engine.Diagnose(ctx, middleware, environment, params)
			}

			if err != nil {
				logging.Logger.Errorf("Diagnosis failed: %v", err)
				fmt.Printf("诊断失败: %v\n", err)
				os.Exit(1)
			}

			// 输出结果。Output result.
			outputFormat, _ := cmd.Flags().GetString("output")
			switch outputFormat {
			case "json":
				jsonOutput, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(jsonOutput))
			default:
				printHumanReadableResult(result)
			}
		},
	}

	// 添加命令行标志。Add command flags.
	cmd.Flags().StringP("environment", "e", "kubernetes", "环境类型 (kubernetes 或 baremetal)。Environment type (kubernetes or baremetal)")
	cmd.Flags().StringP("namespace", "n", "default", "Kubernetes命名空间。Kubernetes namespace")
	cmd.Flags().StringP("labels", "l", "", "标签选择器。Label selector")
	cmd.Flags().StringP("query", "q", "", "自然语言查询。Natural language query")
	cmd.Flags().StringP("output", "o", "human", "输出格式 (human 或 json)。Output format (human or json)")

	return cmd
}

// 以人类可读格式打印诊断结果。Print diagnosis result in human-readable format.
func printHumanReadableResult(result *models.DiagnosisResult) {
	fmt.Printf("=== 诊断结果: %s ===\n", result.DiagnosisID)
	fmt.Printf("中间件类型: %s\n", result.Middleware)
	fmt.Printf("环境: %s\n", result.Environment)
	fmt.Printf("时间: %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Printf("状态: %s\n", result.Status)
	fmt.Printf("诊断耗时: %.2f秒\n", result.Duration)
	fmt.Println("----------------------------------------")

	if len(result.Findings) == 0 {
		fmt.Println("未发现问题。No issues found.")
		return
	}

	for i, finding := range result.Findings {
		fmt.Printf("\n问题 %d: %s\n", i+1, finding.Title)
		fmt.Printf("类型: %s, 严重程度: %s\n", finding.Type, finding.Severity)
		fmt.Println("详细描述:")
		fmt.Println(finding.Detail)

		if len(finding.Evidence) > 0 {
			fmt.Println("证据:")
			for _, evidence := range finding.Evidence {
				fmt.Printf("- %s\n", evidence)
			}
		}

		if len(finding.Recommendations) > 0 {
			fmt.Println("建议解决方案:")
			for j, rec := range finding.Recommendations {
				fmt.Printf("%d. %s\n", j+1, rec.Description)
				if rec.Command != "" {
					fmt.Printf("   命令: %s\n", rec.Command)
				}
				autoFix := "否"
				if rec.AutoFix {
					autoFix = "是"
				}
				fmt.Printf("   可自动修复: %s, 风险等级: %s\n", autoFix, rec.RiskLevel)
			}
		}
	}
}

//Personal.AI order the ending
