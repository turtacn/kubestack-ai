package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// OutputFormat defines supported output formats
type OutputFormat string

const (
	OutputFormatJSON  OutputFormat = "json"
	OutputFormatYAML  OutputFormat = "yaml"
	OutputFormatTable OutputFormat = "table"
	OutputFormatPlain OutputFormat = "plain"
)

// OutputResult outputs data in the specified format
func OutputResult(data interface{}, format string) error {
	switch OutputFormat(format) {
	case OutputFormatJSON:
		return outputJSON(data)
	case OutputFormatYAML:
		return outputYAML(data)
	case OutputFormatTable:
		return outputTable(data)
	default:
		return outputPlain(data)
	}
}

func outputJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func outputYAML(data interface{}) error {
	enc := yaml.NewEncoder(os.Stdout)
	defer enc.Close()
	return enc.Encode(data)
}

func outputTable(data interface{}) error {
	switch v := data.(type) {
	case *plugin.DiagnosticResult:
		return formatDiagnosticResult(v)
	case []plugin.EnhancedPluginInfo:
		return formatPluginList(v)
	case map[string]plugin.EnhancedPluginInfo:
		return formatPluginMap(v)
	default:
		return outputPlain(data)
	}
}

func outputPlain(data interface{}) error {
	fmt.Printf("%+v\n", data)
	return nil
}

func formatDiagnosticResult(result *plugin.DiagnosticResult) error {
	// Header
	fmt.Printf("\n╔═══════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  Diagnostic Report                                                ║\n")
	fmt.Printf("╠═══════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("  Plugin:     %s\n", result.PluginID)
	fmt.Printf("  Target:     %s\n", result.TargetName)
	fmt.Printf("  Status:     %s\n", colorizeStatus(result.Status))
	fmt.Printf("  Duration:   %s\n", result.Duration)
	fmt.Printf("  Timestamp:  %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("╚═══════════════════════════════════════════════════════════════════╝\n\n")
	
	// Findings
	if len(result.Findings) > 0 {
		fmt.Printf("Findings (%d):\n", len(result.Findings))
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		
		for i, finding := range result.Findings {
			fmt.Printf("\n%d. [%s] %s\n", i+1, colorizeSeverity(finding.Severity), finding.Title)
			fmt.Printf("   Category: %s\n", finding.Category)
			if finding.Description != "" {
				fmt.Printf("   %s\n", finding.Description)
			}
			if finding.Remediation != "" {
				fmt.Printf("   💡 Remediation: %s\n", finding.Remediation)
			}
		}
		fmt.Println()
	} else {
		fmt.Println("✓ No issues found")
	}
	
	// Metrics
	if len(result.Metrics) > 0 {
		fmt.Printf("Metrics:\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for key, value := range result.Metrics {
			fmt.Fprintf(tw, "  %s:\t%v\n", key, value)
		}
		tw.Flush()
		fmt.Println()
	}
	
	// Suggestions
	if len(result.Suggestions) > 0 {
		fmt.Printf("Suggestions:\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		for _, suggestion := range result.Suggestions {
			fmt.Printf("  • %s\n", suggestion)
		}
		fmt.Println()
	}
	
	return nil
}

func formatPluginList(plugins []plugin.EnhancedPluginInfo) error {
	if len(plugins) == 0 {
		fmt.Println("No plugins found")
		return nil
	}
	
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tVERSION\tTYPE\tCAPABILITIES")
	fmt.Fprintln(tw, "────────────────\t────────────────\t─────────\t────────────\t─────────────────────")
	
	for _, p := range plugins {
		capabilities := ""
		if len(p.Capabilities) > 0 {
			capabilities = p.Capabilities[0]
			if len(p.Capabilities) > 1 {
				capabilities += fmt.Sprintf(" (+%d)", len(p.Capabilities)-1)
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.ID, p.Name, p.Version, p.Type, capabilities)
	}
	
	return tw.Flush()
}

func formatPluginMap(plugins map[string]plugin.EnhancedPluginInfo) error {
	list := make([]plugin.EnhancedPluginInfo, 0, len(plugins))
	for _, p := range plugins {
		list = append(list, p)
	}
	return formatPluginList(list)
}

func colorizeStatus(status plugin.DiagnosticStatus) string {
	switch status {
	case plugin.DiagnosticStatusHealthy:
		return "\033[32m✓ HEALTHY\033[0m"
	case plugin.DiagnosticStatusWarning:
		return "\033[33m⚠ WARNING\033[0m"
	case plugin.DiagnosticStatusCritical:
		return "\033[31m✗ CRITICAL\033[0m"
	default:
		return "\033[90m? UNKNOWN\033[0m"
	}
}

func colorizeSeverity(severity plugin.Severity) string {
	switch severity {
	case plugin.SeverityInfo:
		return "\033[36mINFO\033[0m"
	case plugin.SeverityWarning:
		return "\033[33mWARN\033[0m"
	case plugin.SeverityError:
		return "\033[31mERROR\033[0m"
	case plugin.SeverityCritical:
		return "\033[91mCRITICAL\033[0m"
	default:
		return "UNKNOWN"
	}
}
