package diagnosis

import (
	"time"

	"github.com/google/uuid"
	"github.com/turtacn/kubestack-ai/internal/ai/llm"
	"github.com/turtacn/kubestack-ai/internal/collectors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
	"github.com/turtacn/kubestack-ai/internal/pluginmgr"
)

// DiagnosisEngine 接口定义诊断引擎。DiagnosisEngine interface for diagnosis.
type DiagnosisEngine interface {
	Diagnose(middleware string) (*models.DiagnosisResult, error)
}

// engine 诊断引擎实现。engine implements DiagnosisEngine.
type engine struct {
	collector collectors.Collector
	pluginMgr pluginmgr.PluginManager
	llm       llm.LLM
}

// NewEngine 创建诊断引擎。NewEngine creates diagnosis engine.
func NewEngine(c collectors.Collector, pm pluginmgr.PluginManager, l llm.LLM) DiagnosisEngine {
	return &engine{collector: c, pluginMgr: pm, llm: l}
}

// Diagnose 执行诊断。Diagnose performs diagnosis.
func (e *engine) Diagnose(middleware string) (*models.DiagnosisResult, error) {
	p, err := e.pluginMgr.Load(middleware)
	if err != nil {
		return nil, err
	}

	// 采集数据。Collect data.
	metrics, _ := p.CollectMetrics()
	logs, _ := p.AnalyzeLogs()
	config, _ := p.ValidateConfig()

	// 调用AI分析。Call AI analysis.
	findings, err := e.llm.Analyze(metrics, logs, config)
	if err != nil {
		logging.Logger.Error(err)
		return nil, err
	}

	return &models.DiagnosisResult{
		DiagnosisID: uuid.New().String(),
		Middleware:  middleware,
		Timestamp:   time.Now(),
		Status:      "healthy", // 示例。Example.
		Findings:    findings,
	}, nil

}

//Personal.AI order the ending
