package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// CommandExecutor Kafka command executor
type CommandExecutor struct {
	client sarama.Client
	admin  sarama.ClusterAdmin
}

func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

func (e *CommandExecutor) SetClient(client sarama.Client, admin sarama.ClusterAdmin) {
	e.client = client
	e.admin = admin
}

func (e *CommandExecutor) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	result := &plugin.CommandResult{}
	startTime := time.Now()

	if cmd.DryRun {
		result.Success = true
		result.Output = fmt.Sprintf("[DRY-RUN] Would execute: %s %v", cmd.Name, cmd.Args)
		return result, nil
	}

	switch strings.ToUpper(cmd.Name) {
	case "LIST TOPICS":
		topics, err := e.client.Topics()
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Output = fmt.Sprintf("Topics: %v", topics)
		}

	case "DELETE TOPIC":
		if len(cmd.Args) != 1 {
			return nil, fmt.Errorf("DELETE TOPIC requires 1 argument (topic name)")
		}
		topicName, ok := cmd.Args[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string")
		}
		err := e.admin.DeleteTopic(topicName)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Output = fmt.Sprintf("Topic %s deleted", topicName)
		}

	default:
		return nil, fmt.Errorf("unsupported command: %s", cmd.Name)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}
