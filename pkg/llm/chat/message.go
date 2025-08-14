package chat

import (
	"encoding/json"
	"fmt"
	"time"
)

// 声明schema的参数类型
type SchemaType string
type MessageType string

const (
	TypeObject SchemaType = "object" // 对象类型
	TypeArray  SchemaType = "array"  // 数组类型

	TypeString  SchemaType = "string"  // 字符串类型
	TypeBoolean SchemaType = "boolean" // 布尔类型
	TypeNumber  SchemaType = "number"  // 数值类型
	TypeInteger SchemaType = "integer" // 整数类型
)

// Message 是发送到LLM/从LLM接收的有效负载。
const (
	MessageTypeText             MessageType = "text"
	MessageTypeToolCallResponse MessageType = "tool-call-response"
)

type Message struct {
	ID            string
	Source        MessageSource
	Type          MessageType
	Payload       string
	FunCallResult *FunctionCallResult
	Timestamp     time.Time
}

type FunctionCallResult struct {
	ID     string         `json:"id,omitempty"`
	Name   string         `json:"name,omitempty"`
	Result map[string]any `json:"result,omitempty"`
}

type MessageSource string

const (
	MessageSourceUser  MessageSource = "user"
	MessageSourceAgent MessageSource = "agent"
	MessageSourceModel MessageSource = "model"
)

// FunctionCall 是对语言模型的函数调用。
// LLM将用FunctionCall回复用户定义的函数，我们将结果返回。
type FunctionCall struct {
	ID        string         `json:"id,omitempty"`        // 调用ID
	Name      string         `json:"name,omitempty"`      // 函数名称
	Arguments map[string]any `json:"arguments,omitempty"` // 参数映射
}

// FunctionDefinition 是LLM可调用的用户定义函数。
// 如果LLM确定应该调用该函数，它将回复一个FunctionCall对象；
// 我们将调用该函数并返回结果。
type FunctionDefinition struct {
	Name        string   `json:"name,omitempty"`        // 函数名称
	Description string   `json:"description,omitempty"` // 函数描述
	Parameters  *Schema  `json:"parameters,omitempty"`  // 参数模式
	Required    []string `json:"required,omitempty"`    // 必需字段
}

// Schema 是为结构化输出指定JSON模式。
type Schema struct {
	Type        string             `json:"type,omitempty"`        // 类型
	Properties  map[string]*Schema `json:"properties,omitempty"`  // 属性映射
	Items       *Schema            `json:"items,omitempty"`       // 数组项模式
	Description string             `json:"description,omitempty"` // 描述
	Required    []string           `json:"required,omitempty"`    // 必需字段
}

// ToRawSchema 将 Schema 转换为 json.RawMessage。
func (s *Schema) ToRawSchema() (json.RawMessage, error) {
	jsonSchema, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("将工具模式转换为json: %w", err)
	}
	var rawSchema json.RawMessage
	if err := json.Unmarshal(jsonSchema, &rawSchema); err != nil {
		return nil, fmt.Errorf("将工具模式转换为json.RawMessage: %w", err)
	}
	return rawSchema, nil
}
