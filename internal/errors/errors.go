package errors

import "errors"

// ErrPluginNotFound 插件未找到错误。ErrPluginNotFound plugin not found error.
var ErrPluginNotFound = errors.New("plugin not found")

// ErrInvalidConfig 配置无效错误。ErrInvalidConfig invalid config error.
var ErrInvalidConfig = errors.New("invalid configuration")

// ErrDiagnosisFailed 诊断失败错误。ErrDiagnosisFailed diagnosis failed error.
var ErrDiagnosisFailed = errors.New("diagnosis failed")

// ErrLLMCallFailed LLM调用失败错误。ErrLLMCallFailed LLM call failed error.
var ErrLLMCallFailed = errors.New("LLM call failed")

// 更多错误定义，根据需求扩展。More error definitions, extend as needed.
//Personal.AI order the ending
