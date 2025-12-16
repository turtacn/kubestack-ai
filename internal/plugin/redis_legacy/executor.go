//go:build redis_legacy
// +build redis_legacy

package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// CommandExecutor Redis command executor
type CommandExecutor struct {
	client      *redis.Client
	blockedCmds map[string]bool
}

func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		blockedCmds: map[string]bool{
			"DEBUG":    true,
			"SHUTDOWN": true,
			"SLAVEOF":  true,
			"REPLICAOF": true,
			"CLUSTER":  true,
		},
	}
}

func (e *CommandExecutor) SetClient(client *redis.Client) {
	e.client = client
}

// Execute executes a command
func (e *CommandExecutor) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	result := &plugin.CommandResult{}
	startTime := time.Now()

	// 1. Security check
	upperName := strings.ToUpper(cmd.Name)
	if e.blockedCmds[upperName] {
		return nil, fmt.Errorf("command %s is blocked for security", cmd.Name)
	}

	// 2. DryRun
	if cmd.DryRun {
		result.Success = true
		result.Output = fmt.Sprintf("[DRY-RUN] Would execute: %s %v", cmd.Name, cmd.Args)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// 3. Timeout
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	// 4. Build and execute
	args := make([]interface{}, len(cmd.Args)+1)
	args[0] = cmd.Name
	for i, arg := range cmd.Args {
		args[i+1] = arg
	}

	redisResult := e.client.Do(ctx, args...)
	result.Duration = time.Since(startTime)

	if err := redisResult.Err(); err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	// 5. Result
	result.Success = true
	result.Output = fmt.Sprintf("%v", redisResult.Val())

	return result, nil
}
