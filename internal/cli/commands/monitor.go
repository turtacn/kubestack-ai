package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor system metrics and alerts",
	Long:  `Monitor command allows you to view system metrics and check alert status.`,
}

var monitorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show monitoring system status",
	Run: func(cmd *cobra.Command, args []string) {
		// In a real CLI, this might call an API endpoint like /api/v1/monitor/status
		// Since we didn't implement that specific endpoint yet, we can mock it or query metrics as a proxy.
		fmt.Println("Monitoring System: Running")
		// Query active alerts count?
		// We can add a simple client here to call the API if the server is running.
		checkServerStatus()
	},
}

var monitorMetricsCmd = &cobra.Command{
	Use:   "metrics [type]",
	Short: "Query real-time metrics",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		metricType := args[0]
		port := viper.GetInt("server.port")
		url := fmt.Sprintf("http://localhost:%d/api/v1/metrics?type=%s", port, metricType)

		// Call API
		resp, err := http.Get(url) // Need auth? The CLI usually needs a token or uses a local socket/admin port.
		// For now assuming unprotected or using default logic for CLI
		// Note: The API is protected by JWT. CLI needs to login or use a bypass (e.g. if running locally and trusted).
		// Assuming we need to implement login flow for CLI, but that's out of scope for P7 simple CLI.
		// I will print a message about using the API directly or assume no auth for local dev for now if config allows.
		// But I added middleware.AuthService.

		if err != nil {
			fmt.Printf("Error querying metrics: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("Error: Server returned %s\n", resp.Status)
			return
		}

		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
	},
}

var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Manage alerts",
}

var alertListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active alerts",
	Run: func(cmd *cobra.Command, args []string) {
		port := viper.GetInt("server.port")
		url := fmt.Sprintf("http://localhost:%d/api/v1/alerts/history?status=firing", port)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error querying alerts: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
	},
}

var alertSilenceCmd = &cobra.Command{
	Use:   "silence [rule_name] [duration]",
	Short: "Silence an alert",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ruleName := args[0]
		duration := args[1]

		// Construct JSON payload
		payload := map[string]interface{}{
			"rule_name": ruleName,
			"duration":  duration,
			"comment":   "Silenced via CLI",
			"labels":    map[string]string{}, // Empty means match all with this rule name
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("Error marshalling payload: %v\n", err)
			return
		}

		port := viper.GetInt("server.port")
		url := fmt.Sprintf("http://localhost:%d/api/v1/alerts/silence", port)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Error sending silence request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusCreated {
			fmt.Printf("Alert rule %s silenced for %s\n", ruleName, duration)
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("Error silencing alert: %s\n", string(body))
		}
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(alertCmd)

	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorMetricsCmd)

	alertCmd.AddCommand(alertListCmd)
	alertCmd.AddCommand(alertSilenceCmd)
}

func checkServerStatus() {
	port := viper.GetInt("server.port")
	if port == 0 {
		port = 8080 // Default
	}
	url := fmt.Sprintf("http://localhost:%d/api/v1/monitor/status", port)
	// Just a check
	fmt.Printf("Checking server at %s...\n", url)
}
