package errors

import "errors"

// ErrPluginNotFound 插件未找到错误。ErrPluginNotFound plugin not found error.
var ErrPluginNotFound = errors.New("plugin not found")

// ErrInvalidConfig 配置无效错误。ErrInvalidConfig invalid configuration error.
var ErrInvalidConfig = errors.New("invalid configuration")

// ErrDiagnosisFailed 诊断失败错误。ErrDiagnosisFailed diagnosis failed error.
var ErrDiagnosisFailed = errors.New("diagnosis failed")

// ErrLLMCallFailed LLM调用失败错误。ErrLLMCallFailed LLM call failed error.
var ErrLLMCallFailed = errors.New("LLM call failed")

// ErrDataCollectionFailed 数据收集失败错误。ErrDataCollectionFailed data collection failed error.
var ErrDataCollectionFailed = errors.New("data collection failed")

// ErrRAGRetrievalFailed RAG检索失败错误。ErrRAGRetrievalFailed RAG retrieval failed error.
var ErrRAGRetrievalFailed = errors.New("RAG retrieval failed")

// ErrPluginInstallationFailed 插件安装失败错误。ErrPluginInstallationFailed plugin installation failed error.
var ErrPluginInstallationFailed = errors.New("plugin installation failed")

// ErrKubernetesClientFailed Kubernetes客户端初始化失败。ErrKubernetesClientFailed Kubernetes client initialization failed.
var ErrKubernetesClientFailed = errors.New("Kubernetes client initialization failed")

//Personal.AI order the ending
