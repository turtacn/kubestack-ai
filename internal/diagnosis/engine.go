package diagnosis

import (
	"context"
	"time"

	"github.com/turtacn/kubestack-ai/internal/ai"
	"github.com/turtacn/kubestack-ai/internal/collectors"
	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// DiagnosisEngine 接口定义诊断引擎。DiagnosisEngine interface for diagnosis.
type DiagnosisEngine interface {
	Diagnose(ctx context.Context, middleware string, environment string, params map[string]string) (*models.DiagnosisResult, error)
	DiagnoseWithQuery(ctx context.Context, middleware string, query string) (*models.DiagnosisResult, error)
	GetDiagnosisHistory() []*models.DiagnosisResult
}

// engine 诊断引擎实现。engine implements DiagnosisEngine.
type engine struct {
	collector   collectors.Collector
	pluginMgr   plugins.PluginManager
	llm         ai.LLM
	rag         ai.RAG
	history     []*models.DiagnosisResult
	historyLock sync.RWMutex
}

// NewEngine 创建诊断引擎。NewEngine creates diagnosis engine.
func NewEngine(c collectors.Collector, pm plugins.PluginManager, l ai.LLM, rag ai.RAG) DiagnosisEngine {
	return &engine{
		collector: c,
		pluginMgr: pm,
		llm:       l,
		rag:       rag,
		history:   []*models.DiagnosisResult{},
	}
}

// Diagnose 执行诊断。Diagnose performs diagnosis.
func (e *engine) Diagnose(ctx context.Context, middleware string, environment string, params map[string]string) (*models.DiagnosisResult, error) {
	startTime := time.Now()
	logging.Logger.Infof("Starting diagnosis for %s in %s environment", middleware, environment)

	// 1. 加载相应的插件。1. Load appropriate plugin.
	plugin, err := e.pluginMgr.Load(middleware)
	if err != nil {
		logging.Logger.Errorf("Failed to load plugin %s: %v", middleware, err)
		return nil, errors.ErrDiagnosisFailed
	}

	// 2. 收集通用环境数据。2. Collect general environment data.
	envInfo, err := e.collector.GetEnvironmentInfo(ctx)
	if err != nil {
		logging.Logger.Warnf("Failed to collect environment info: %v", err)
	}

	// 3. 收集中间件特定数据。3. Collect middleware-specific data.
	metrics, err := plugin.CollectMetrics(ctx)
	if err != nil {
		logging.Logger.Errorf("Failed to collect metrics: %v", err)
		return nil, errors.ErrDataCollectionFailed
	}

	logs, err := plugin.CollectLogs(ctx)
	if err != nil {
		logging.Logger.Warnf("Failed to collect logs: %v", err)
	}

	config, err := plugin.CollectConfig(ctx)
	if err != nil {
		logging.Logger.Warnf("Failed to collect config: %v", err)
	}

	// 4. 从RAG获取相关知识。4. Retrieve relevant knowledge from RAG.
	knowledgeQuery := "diagnose " + middleware + " issues with metrics and logs"
	knowledge, err := e.rag.Retrieve(knowledgeQuery, middleware)
	if err != nil {
		logging.Logger.Warnf("Failed to retrieve knowledge: %v", err)
	}

	// 5. 调用AI分析。5. Call AI analysis.
	findings, err := e.llm.Analyze(ctx, metrics, logs, config, knowledge, envInfo)
	if err != nil {
		logging.Logger.Errorf("AI analysis failed: %v", err)
		return nil, errors.ErrLLMCallFailed
	}

	// 6. 生成诊断结果。6. Generate diagnosis result.
	result := models.NewDiagnosisResult(middleware, environment)
	result.Findings = findings
	result.Duration = time.Since(startTime).Seconds()

	// 7. 确定整体状态。7. Determine overall status.
	result.Status = determineOverallStatus(findings)

	// 8. 保存到历史记录。8. Save to history.
	e.historyLock.Lock()
	e.history = append(e.history, result)
	// 限制历史记录大小。Limit history size.
	if len(e.history) > 100 {
		e.history = e.history[1:]
	}
	e.historyLock.Unlock()

	logging.Logger.Infof("Diagnosis completed for %s in %.2f seconds. Status: %s",
		middleware, result.Duration, result.Status)

	return result, nil
}

// DiagnoseWithQuery 基于自然语言查询执行诊断。Diagnose with natural language query.
func (e *engine) DiagnoseWithQuery(ctx context.Context, middleware string, query string) (*models.DiagnosisResult, error) {
	logging.Logger.Infof("Diagnosing %s with query: %s", middleware, query)

	// 获取相关知识。Get relevant knowledge.
	knowledge, err := e.rag.Retrieve(query, middleware)
	if err != nil {
		logging.Logger.Warnf("Failed to retrieve knowledge for query: %v", err)
	}

	// 调用LLM理解查询意图。Call LLM to understand query intent.
	intentAnalysis, err := e.llm.Query(ctx, "Analyze this query to determine what needs to be checked: "+query+
		". Return specific metrics, logs, and configurations to examine.")
	if err != nil {
		logging.Logger.Errorf("Failed to analyze query intent: %v", err)
		return nil, errors.ErrLLMCallFailed
	}

	logging.Logger.Debugf("Query intent analysis: %s", intentAnalysis)

	// 执行标准诊断并附加查询信息。Perform standard diagnosis with query info.
	params := map[string]string{
		"query":           query,
		"intent_analysis": intentAnalysis,
	}
	return e.Diagnose(ctx, middleware, "", params)
}

// GetDiagnosisHistory 获取诊断历史。Get diagnosis history.
func (e *engine) GetDiagnosisHistory() []*models.DiagnosisResult {
	e.historyLock.RLock()
	defer e.historyLock.RUnlock()

	// 返回历史记录的副本。Return copy of history.
	historyCopy := make([]*models.DiagnosisResult, len(e.history))
	copy(historyCopy, e.history)
	return historyCopy
}

// determineOverallStatus 根据发现的问题确定整体状态。Determine overall status based on findings.
func determineOverallStatus(findings []models.Finding) string {
	for _, finding := range findings {
		if finding.Severity == "high" {
			return "critical"
		}
	}

	for _, finding := range findings {
		if finding.Severity == "medium" {
			return "warning"
		}
	}

	return "healthy"
}

//Personal.AI order the ending
