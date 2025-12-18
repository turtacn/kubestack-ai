// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// KBEntry represents a knowledge base entry
type KBEntry struct {
	ID          string    `json:"id" yaml:"id"`
	Title       string    `json:"title" yaml:"title"`
	Severity    string    `json:"severity" yaml:"severity"`
	Middleware  string    `json:"middleware" yaml:"middleware"`
	Created     time.Time `json:"created" yaml:"created"`
	Updated     time.Time `json:"updated" yaml:"updated"`
	Content     string    `json:"content" yaml:"content"`
	Summary     string    `json:"summary" yaml:"summary"`
	Tags        []string  `json:"tags" yaml:"tags"`
	RelatedDocs []string  `json:"related_docs" yaml:"related_docs"`
}

// newKBCmd creates the kb command for knowledge base operations
func newKBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kb",
		Short: "Knowledge base search and management",
		Long: `Search and manage the KubeStack-AI knowledge base.
The knowledge base contains diagnostic procedures, best practices,
troubleshooting guides, and solutions for various middleware issues.`,
		Example: `  # Search knowledge base
  ksa kb search "Redis OOM"

  # Get entry details
  ksa kb get kb-redis-001

  # Update knowledge base
  ksa kb update`,
	}

	cmd.AddCommand(newKBSearchCmd())
	cmd.AddCommand(newKBGetCmd())
	cmd.AddCommand(newKBUpdateCmd())

	return cmd
}

// newKBSearchCmd creates the kb search subcommand
func newKBSearchCmd() *cobra.Command {
	var (
		severity   string
		middleware string
		limit      int
		full       bool
	)

	cmd := &cobra.Command{
		Use:   "search <keyword>",
		Short: "Search the knowledge base",
		Long: `Search the knowledge base for entries matching the given keyword.
Results can be filtered by severity, middleware type, and limited in number.`,
		Example: `  # Basic search
  ksa kb search "OOM"

  # Search with filters
  ksa kb search "memory" --middleware redis --severity critical

  # Get full content
  ksa kb search "Redis" --full --limit 5

  # JSON output
  ksa kb search "performance" -o json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyword := strings.Join(args, " ")
			
			// Get output format from flag
			outputFormat, _ := cmd.Flags().GetString("output")

			// Get knowledge base client
			kb := getKnowledgeBase()

			// Search with filters
			results, err := kb.Search(context.Background(), keyword, severity, middleware, limit, full)
			if err != nil {
				return fmt.Errorf("failed to search knowledge base: %w", err)
			}

			// Output results
			if outputFormat == "json" {
				return kbOutputJSON(results)
			} else if outputFormat == "yaml" {
				return kbOutputYAML(results)
			} else if outputFormat == "table" {
				return outputKBSearchTable(results, full)
			}

			// Default text output
			return outputKBSearchText(results, full)
		},
	}

	cmd.Flags().StringVar(&severity, "severity", "", "Filter by severity (critical, warning, info)")
	cmd.Flags().StringVar(&middleware, "middleware", "", "Filter by middleware type (redis, mysql, kafka, etc)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of results to return")
	cmd.Flags().BoolVar(&full, "full", false, "Return full entry content (default is summary only)")

	return cmd
}

// newKBGetCmd creates the kb get subcommand
func newKBGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <entry-id>",
		Short: "Get knowledge base entry details",
		Long: `Retrieve and display detailed information about a specific knowledge base entry.
The entry ID can be obtained from search results.`,
		Example: `  # Get entry details
  ksa kb get kb-redis-001

  # Get in JSON format
  ksa kb get kb-redis-001 -o json

  # Get in YAML format
  ksa kb get kb-redis-001 -o yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entryID := args[0]
			
			// Get output format from flag
			outputFormat, _ := cmd.Flags().GetString("output")

			// Get knowledge base client
			kb := getKnowledgeBase()

			// Get entry
			entry, err := kb.Get(context.Background(), entryID)
			if err != nil {
				return fmt.Errorf("failed to get entry: %w", err)
			}

			// Output result
			if outputFormat == "json" {
				return kbOutputJSON(entry)
			} else if outputFormat == "yaml" {
				return kbOutputYAML(entry)
			}

			// Default text output
			return outputKBEntryText(entry)
		},
	}

	return cmd
}

// newKBUpdateCmd creates the kb update subcommand
func newKBUpdateCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the knowledge base",
		Long: `Synchronize the local knowledge base with remote sources.
This will download the latest entries and update existing ones.`,
		Example: `  # Update knowledge base
  ksa kb update

  # Force update (overwrite local changes)
  ksa kb update --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get knowledge base client
			kb := getKnowledgeBase()

			fmt.Println("Updating knowledge base...")

			// Update
			stats, err := kb.Update(context.Background(), force)
			if err != nil {
				return fmt.Errorf("failed to update knowledge base: %w", err)
			}

			// Output stats
			fmt.Printf("Knowledge base updated successfully\n")
			fmt.Printf("  New entries: %d\n", stats["new"])
			fmt.Printf("  Updated entries: %d\n", stats["updated"])
			fmt.Printf("  Deleted entries: %d\n", stats["deleted"])
			fmt.Printf("  Total entries: %d\n", stats["total"])

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force update, overwriting local changes")

	return cmd
}

// KnowledgeBaseClient interface for knowledge base operations
type KnowledgeBaseClient interface {
	Search(ctx context.Context, keyword, severity, middleware string, limit int, full bool) ([]*KBEntry, error)
	Get(ctx context.Context, entryID string) (*KBEntry, error)
	Update(ctx context.Context, force bool) (map[string]int, error)
}

// mockKBClient implements KnowledgeBaseClient for demo purposes
type mockKBClient struct{}

func (m *mockKBClient) Search(ctx context.Context, keyword, severity, middleware string, limit int, full bool) ([]*KBEntry, error) {
	// Mock data for demonstration
	entries := []*KBEntry{
		{
			ID:         "kb-redis-001",
			Title:      "Redis OOM 紧急处理方案",
			Severity:   "critical",
			Middleware: "redis",
			Created:    time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
			Updated:    time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC),
			Summary:    "Redis 内存溢出（OOM）的紧急处理步骤和根因分析方法",
			Content: `1. 临时缓解措施：
   a. 登录 Redis 执行 CONFIG GET maxmemory 确认内存上限；
   b. 执行 CONFIG SET maxmemory-policy allkeys-lru 开启键淘汰；
   c. 使用 redis-cli --bigkeys 定位大键，分批删除；
   
2. 根因排查：
   a. 检查是否有内存泄漏（对比 used_memory 趋势）；
   b. 确认过期键是否未及时清理（查看 expired_keys 指标）；
   c. 分析慢查询日志，识别异常命令；
   
3. 长期优化：
   a. 配置合理的 maxmemory 和淘汰策略；
   b. 启用 RDB/AOF 持久化策略；
   c. 实施键命名规范和过期时间管理；
   d. 考虑 Redis Cluster 或分片方案。`,
			Tags:        []string{"redis", "oom", "memory", "emergency"},
			RelatedDocs: []string{"kb-redis-002", "kb-redis-005"},
		},
		{
			ID:         "kb-redis-002",
			Title:      "Redis 内存优化最佳实践",
			Severity:   "info",
			Middleware: "redis",
			Created:    time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC),
			Updated:    time.Date(2025, 10, 10, 0, 0, 0, 0, time.UTC),
			Summary:    "Redis 内存使用优化的最佳实践和配置建议",
			Content: `1. 数据结构优化：
   a. 使用 Hash 代替 String 存储对象；
   b. 设置合理的 hash-max-ziplist-entries；
   c. 避免使用大 key，拆分为多个小 key；
   
2. 内存策略配置：
   a. 设置 maxmemory 为物理内存的 75%；
   b. 选择合适的淘汰策略（推荐 allkeys-lru）；
   c. 启用 activedefrag yes 进行内存碎片整理；
   
3. 持久化优化：
   a. 使用 AOF everysec 平衡性能和安全；
   b. 配置 save 策略避免频繁 BGSAVE；
   c. 监控 rdb_last_save_time 确保备份正常。`,
			Tags:        []string{"redis", "optimization", "memory", "best-practice"},
			RelatedDocs: []string{"kb-redis-001", "kb-redis-003"},
		},
		{
			ID:         "kb-mysql-001",
			Title:      "MySQL 慢查询优化指南",
			Severity:   "warning",
			Middleware: "mysql",
			Created:    time.Date(2025, 9, 5, 0, 0, 0, 0, time.UTC),
			Updated:    time.Date(2025, 10, 12, 0, 0, 0, 0, time.UTC),
			Summary:    "MySQL 慢查询的识别、分析和优化方法",
			Content: `1. 慢查询识别：
   a. 检查 slow_query_log 是否开启；
   b. 设置合理的 long_query_time（建议 1-2s）；
   c. 使用 pt-query-digest 分析慢查询日志；
   
2. 查询优化：
   a. 使用 EXPLAIN 分析执行计划；
   b. 添加合适的索引（避免全表扫描）；
   c. 重写复杂查询，避免子查询；
   
3. 索引优化：
   a. 遵循最左前缀原则；
   b. 避免在索引列使用函数；
   c. 定期 ANALYZE TABLE 更新统计信息。`,
			Tags:        []string{"mysql", "slow-query", "optimization", "index"},
			RelatedDocs: []string{"kb-mysql-002", "kb-mysql-004"},
		},
	}

	// Filter by keyword
	var filtered []*KBEntry
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Title), strings.ToLower(keyword)) ||
			strings.Contains(strings.ToLower(entry.Content), strings.ToLower(keyword)) ||
			strings.Contains(strings.ToLower(entry.Summary), strings.ToLower(keyword)) {
			
			// Apply filters
			if severity != "" && entry.Severity != severity {
				continue
			}
			if middleware != "" && entry.Middleware != middleware {
				continue
			}
			
			filtered = append(filtered, entry)
		}
	}

	// Apply limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

func (m *mockKBClient) Get(ctx context.Context, entryID string) (*KBEntry, error) {
	// Mock implementation - in real implementation, fetch from database
	entries, _ := m.Search(ctx, "", "", "", 100, true)
	for _, entry := range entries {
		if entry.ID == entryID {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("entry not found: %s", entryID)
}

func (m *mockKBClient) Update(ctx context.Context, force bool) (map[string]int, error) {
	// Mock implementation - simulate update
	time.Sleep(time.Second)
	return map[string]int{
		"new":     5,
		"updated": 12,
		"deleted": 2,
		"total":   145,
	}, nil
}

// getKnowledgeBase returns a knowledge base client
func getKnowledgeBase() KnowledgeBaseClient {
	// TODO: In production, return real KB client based on config
	// For now, return mock client for demonstration
	return &mockKBClient{}
}

// kbOutputJSON outputs data in JSON format
func kbOutputJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// kbOutputYAML outputs data in YAML format
func kbOutputYAML(data interface{}) error {
	enc := yaml.NewEncoder(os.Stdout)
	defer enc.Close()
	return enc.Encode(data)
}

// truncateString truncates a string to the specified length
func truncateKBString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// outputKBSearchTable outputs search results in table format
func outputKBSearchTable(entries []*KBEntry, full bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	if full {
		fmt.Fprintln(w, "ID\tTITLE\tSEVERITY\tMIDDLEWARE\tUPDATED")
		fmt.Fprintln(w, "---\t-----\t--------\t----------\t-------")
	} else {
		fmt.Fprintln(w, "ENTRY ID\tTITLE\tSEVERITY\tMIDDLEWARE")
		fmt.Fprintln(w, "--------\t-----\t--------\t----------")
	}

	for _, entry := range entries {
		if full {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				entry.ID,
				truncateKBString(entry.Title, 40),
				entry.Severity,
				entry.Middleware,
				entry.Updated.Format("2006-01-02"))
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				entry.ID,
				truncateKBString(entry.Title, 40),
				entry.Severity,
				entry.Middleware)
		}
	}

	w.Flush()
	return nil
}

// outputKBSearchText outputs search results in text format
func outputKBSearchText(entries []*KBEntry, full bool) error {
	if len(entries) == 0 {
		fmt.Println("No entries found")
		return nil
	}

	fmt.Printf("Found %d entries:\n\n", len(entries))

	for i, entry := range entries {
		fmt.Printf("[%d] %s\n", i+1, entry.Title)
		fmt.Printf("    ID: %s\n", entry.ID)
		fmt.Printf("    Severity: %s | Middleware: %s | Updated: %s\n",
			entry.Severity, entry.Middleware, entry.Updated.Format("2006-01-02"))
		
		if full {
			fmt.Printf("    Content:\n%s\n", indentText(entry.Content, 6))
		} else {
			fmt.Printf("    Summary: %s\n", entry.Summary)
		}
		
		if i < len(entries)-1 {
			fmt.Println()
		}
	}

	return nil
}

// outputKBEntryText outputs a single entry in text format
func outputKBEntryText(entry *KBEntry) error {
	fmt.Printf("ID: %s\n", entry.ID)
	fmt.Printf("Title: %s\n", entry.Title)
	fmt.Printf("Severity: %s\n", entry.Severity)
	fmt.Printf("Middleware: %s\n", entry.Middleware)
	fmt.Printf("Created: %s\n", entry.Created.Format("2006-01-02"))
	fmt.Printf("Updated: %s\n", entry.Updated.Format("2006-01-02"))
	
	if len(entry.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(entry.Tags, ", "))
	}
	
	fmt.Printf("\nContent:\n%s\n", entry.Content)
	
	if len(entry.RelatedDocs) > 0 {
		fmt.Printf("\nRelated Documents:\n")
		for _, doc := range entry.RelatedDocs {
			fmt.Printf("  - %s\n", doc)
		}
	}

	return nil
}

// indentText indents all lines of text by the specified number of spaces
func indentText(text string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

func init() {
	// KB command will be registered in root.go
}
