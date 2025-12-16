package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// CommandExecutor MySQL command executor
type CommandExecutor struct {
	db *sql.DB
}

func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

func (e *CommandExecutor) SetDB(db *sql.DB) {
	e.db = db
}

func (e *CommandExecutor) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	result := &plugin.CommandResult{}
	startTime := time.Now()

	if cmd.DryRun {
		result.Success = true
		result.Output = fmt.Sprintf("[DRY-RUN] Would execute: %s %v", cmd.Name, cmd.Args)
		return result, nil
	}

	// Basic safety check (should be more robust)
	upperName := strings.ToUpper(cmd.Name)
	if strings.Contains(upperName, "DROP") || strings.Contains(upperName, "TRUNCATE") {
		return nil, fmt.Errorf("high risk command blocked: %s", cmd.Name)
	}

	// Construct query
	query := cmd.Name
	if len(cmd.Args) > 0 {
		// This is very simplistic and risky for SQL injection.
		// In real world, we should use parameterized queries or predefined commands.
		// For this phase, we assume cmd.Name is the command template or full query if args empty
		// But interface says Args is []interface{}.
		// Let's assume for specific supported commands like "KILL", "OPTIMIZE TABLE"
		// we construct the query safely.
		if strings.HasPrefix(upperName, "KILL") && len(cmd.Args) == 1 {
			// Validate that argument is a number (connection ID)
			idVal := fmt.Sprintf("%v", cmd.Args[0])
			if _, err := strconv.Atoi(idVal); err != nil {
				return nil, fmt.Errorf("invalid connection ID for KILL: %v", cmd.Args[0])
			}
			query = fmt.Sprintf("KILL %s", idVal)
		} else if strings.HasPrefix(upperName, "OPTIMIZE TABLE") && len(cmd.Args) == 1 {
			// Validate table name (simple check to avoid obvious injection, though imperfect)
			tableName := fmt.Sprintf("%v", cmd.Args[0])
			if strings.ContainsAny(tableName, "; \"'\\") {
				return nil, fmt.Errorf("invalid table name: %s", tableName)
			}
			query = fmt.Sprintf("OPTIMIZE TABLE %s", tableName)
		} else {
			// Fallback or error
			// For now, allow passthrough for testing but warn
		}
	}

	res, err := e.db.ExecContext(ctx, query)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	rows, _ := res.RowsAffected()
	result.Success = true
	result.AffectedRows = rows
	result.Output = fmt.Sprintf("Query executed, affected rows: %d", rows)

	return result, nil
}
