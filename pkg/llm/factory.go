package llm

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/turtacn/kubestack-ai/pkg/llm/chat"
	"k8s.io/klog/v2"
)

var globalRegistry registry

func Init() {
	InitOpenai()
}

type registry struct {
	mutex     sync.Mutex
	providers map[string]FactoryFunc
}

func (r *registry) listProviders() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	providers := make([]string, 0, len(r.providers))
	for k := range r.providers {
		providers = append(providers, k)
	}
	return providers
}

type ClientOptions struct {
	URL           *url.URL
	SkipVerifySSL bool
	Headers       map[string]string
	// 根据需要扩展更多选项
}

// Option 是一个用于配置 ClientOptions 的函数式选项。
type Option func(*ClientOptions)

// WithSkipVerifySSL 启用跳过 HTTP 客户端的 SSL 证书验证。
func WithSkipVerifySSL() Option {
	return func(o *ClientOptions) {
		o.SkipVerifySSL = true
	}
}

// WithHeaders 设置 HTTP 客户端的自定义头部。
func WithHeaders(headers map[string]string) Option {
	return func(o *ClientOptions) {
		o.Headers = headers
	}
}

type FactoryFunc func(ctx context.Context, opts ClientOptions) (Client, error)

func RegisterProvider(id string, factoryFunc FactoryFunc) error {
	return globalRegistry.RegisterProvider(id, factoryFunc)
}

func (r *registry) RegisterProvider(id string, factoryFunc FactoryFunc) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.providers == nil {
		r.providers = make(map[string]FactoryFunc)
	}
	_, exists := r.providers[id]
	if exists {
		return fmt.Errorf("provider %q is already registered", id)
	}
	r.providers[id] = factoryFunc
	return nil
}

func (r *registry) NewClient(ctx context.Context, providerID string, opts ...Option) (Client, error) {
	// providerID can be just an ID, for example "gemini" instead of "gemini://"
	if !strings.Contains(providerID, "/") && !strings.Contains(providerID, ":") {
		providerID = providerID + "://"
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	u, err := url.Parse(providerID)
	if err != nil {
		return nil, fmt.Errorf("parsing provider id %q: %w", providerID, err)
	}

	factoryFunc := r.providers[u.Scheme]
	if factoryFunc == nil {
		return nil, fmt.Errorf("provider %q not registered. Available providers: %v", u.Scheme, r.listProviders())
	}

	// Build ClientOptions
	clientOpts := ClientOptions{
		URL:     u,
		Headers: make(map[string]string),
	}
	// Support environment variable override for SkipVerifySSL
	if v := os.Getenv("LLM_SKIP_VERIFY_SSL"); v == "1" || strings.ToLower(v) == "true" {
		clientOpts.SkipVerifySSL = true
	}
	for _, opt := range opts {
		opt(&clientOpts)
	}

	return factoryFunc(ctx, clientOpts)
}

/*
NewClient 根据 LLM_CLIENT 环境变量或提供的 providerID 构建 Client。
如果 providerID 不为空，它会覆盖 LLM_CLIENT 中的值。
支持 Option 参数和 LLM_SKIP_VERIFY_SSL 环境变量。
*/
func NewClient(ctx context.Context, providerID string, opts ...Option) (Client, error) {
	if providerID == "" {
		s := os.Getenv("LLM_CLIENT")
		if s == "" {
			return nil, fmt.Errorf("LLM_CLIENT 未设置。可用提供商：%v", globalRegistry.listProviders())
		}
		providerID = s
	}

	return globalRegistry.NewClient(ctx, providerID, opts...)
}

// APIError 表示 LLM 客户端返回的错误。
type APIError struct {
	StatusCode int    // 状态码
	Message    string // 错误消息
	Err        error  // 原始错误
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("API Error: Status=%d, Message='%s', OriginalErr=%v", e.StatusCode, e.Message, e.Err)
	}
	return fmt.Sprintf("API Error: Status=%d, Message='%s'", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// 判断异常是否可以重试，每个模型的判断标准都不一样，需要根据具体模型来实现
type IsRetryableFunc func(error) bool

// 默认的重试异常判断函数，可以根据需要进行自定义
func DefaultIsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusConflict, http.StatusTooManyRequests,
			http.StatusInternalServerError, http.StatusBadGateway,
			http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}

func createCustomHTTPClient(opts ClientOptions) *http.Client {
	var transport http.RoundTripper

	if opts.SkipVerifySSL {
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	} else {
		transport = http.DefaultTransport
	}

	if len(opts.Headers) == 0 {
		return &http.Client{
			Transport: transport,
		}
	}

	headerTransport := &headerTransport{
		base:    transport,
		headers: opts.Headers,
	}

	return &http.Client{
		Transport: headerTransport,
	}
}

// headerTransport is a custom RoundTripper that adds headers to requests
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a new request to avoid modifying the original
	newReq := req.Clone(req.Context())

	// Add custom headers
	for key, value := range t.headers {
		newReq.Header.Set(key, value)
	}

	// Use the base transport to execute the request
	return t.base.RoundTrip(newReq)
}

// RetryConfig 保存重试机制的配置（与之前相同）
type RetryConfig struct {
	MaxAttempts    int           // 最大尝试次数
	InitialBackoff time.Duration // 初始退避时间
	MaxBackoff     time.Duration // 最大退避时间
	BackoffFactor  float64       // 退避因子
	Jitter         bool          // 是否启用抖动
}

// DefaultRetryConfig 提供合理的默认值（与之前相同）
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:    5,
	InitialBackoff: 200 * time.Millisecond, // 略微增加的默认值
	MaxBackoff:     10 * time.Second,
	BackoffFactor:  2.0,
	Jitter:         true,
}

// Retry 执行提供的操作并带有重试机制，返回结果和错误。
// 现在是泛型的，可以处理任何返回类型 T。
func Retry[T any](
	ctx context.Context,
	config RetryConfig,
	isRetryable IsRetryableFunc,
	operation func(ctx context.Context) (T, error),
) (T, error) {
	var lastErr error
	var zero T // 返回类型 T 的零值

	log := klog.FromContext(ctx)

	backoff := config.InitialBackoff

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		log.V(2).Info("重试尝试开始", "attempt", attempt, "maxAttempts", config.MaxAttempts, "backoff", backoff)
		result, err := operation(ctx)

		if err == nil {
			log.V(2).Info("重试尝试成功", "attempt", attempt)
			return result, nil
		}
		lastErr = err // 存储遇到的最后一个错误

		// 检查操作后上下文是否被取消
		select {
		case <-ctx.Done():
			log.Info("上下文在尝试 %d 失败后被取消。", "attempt", attempt)
			return zero, ctx.Err() // 优先返回上下文错误
		default:
			// 上下文未被取消，继续错误检查
		}

		if !isRetryable(lastErr) {
			log.Info("尝试失败，错误不可重试", "attempt", attempt, "error", lastErr)
			return zero, lastErr // 立即返回不可重试的错误
		}

		log.Info("尝试失败，错误可重试", "attempt", attempt, "error", lastErr)

		if attempt == config.MaxAttempts {
			// 达到最大尝试次数
			break
		}

		// 计算等待时间
		waitTime := backoff
		if config.Jitter {
			waitTime += time.Duration(rand.Float64() * float64(backoff) / 2)
		}

		log.V(2).Info("等待下一次重试尝试", "waitTime", waitTime, "nextAttempt", attempt+1, "maxAttempts", config.MaxAttempts)

		// 等待或响应上下文取消
		select {
		case <-time.After(waitTime):
			// 等待完成
		case <-ctx.Done():
			log.Info("在尝试 %d 后等待重试时上下文被取消。", "attempt", attempt)
			return zero, ctx.Err()
		}

		// 增加退避时间
		backoff = time.Duration(float64(backoff) * config.BackoffFactor)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}
	}

	// 如果循环结束，意味着所有尝试都失败了
	errFinal := fmt.Errorf("操作在 %d 次尝试后失败：%w", config.MaxAttempts, lastErr)
	return zero, errFinal
}

// retryChat 是一个通用装饰器，为任何 Chat 实现添加重试逻辑。
type retryChat[C chat.Chat] struct {
	underlying  chat.Chat // 被包装的实际客户端实现
	config      RetryConfig
	isRetryable IsRetryableFunc
}

// 重试机制的装饰器
func NewRetryChat[C chat.Chat](
	underlying C,
	config RetryConfig,
) chat.Chat {
	return &retryChat[C]{
		underlying: underlying,
		config:     config,
	}
}

// Send 实现了 retryClient 装饰器的 Client 接口。
func (rc *retryChat[C]) Send(ctx context.Context, contents ...*chat.Message) (chat.ChatResponse, error) {
	// 定义操作
	operation := func(ctx context.Context) (chat.ChatResponse, error) {
		return rc.underlying.Send(ctx, contents...)
	}

	// 使用重试机制执行
	return Retry[chat.ChatResponse](ctx, rc.config, rc.underlying.IsRetryableError, operation)
}

// SendStreaming 实现了 retryClient 装饰器的 Client 接口。
func (rc *retryChat[C]) SendStreaming(ctx context.Context, contents ...*chat.Message) (chat.ChatResponseIterator, error) {
	return rc.underlying.SendStreaming(ctx, contents...)
}

func (rc *retryChat[C]) SetFunctionDefinitions(functionDefinitions []*chat.FunctionDefinition) error {
	return rc.underlying.SetFunctionDefinitions(functionDefinitions)
}

func (rc *retryChat[C]) IsRetryableError(err error) bool {
	return rc.underlying.IsRetryableError(err)
}

func (rc *retryChat[C]) Initialize(messages []*chat.Message) error {
	return rc.underlying.Initialize(messages)
}
