#!/bin/bash
#
# scripts/ksa-demo.sh
# Demo runner for kubestack-ai (ksa)
# This script simulates ksa CLI usage with bilingual output (English + 中文)
# Showcases help menu, multi-middleware diagnosis, RCA, and safe fix application
# Usage: ./scripts/ksa-demo.sh
#

# --- Colors ---
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
CYAN="\033[36m"
BOLD="\033[1m"
RESET="\033[0m"

# --- Helper: Simulated progress ---
progress() {
  msg=$1
  echo -ne "${CYAN}${msg}${RESET}"
  for i in {1..3}; do
    echo -ne "."
    sleep 0.4
  done
  echo ""
}

# --- Show simulated help ---
echo -e "${BOLD}${GREEN}>>> Task: ksa --help${RESET}"
cat <<EOF
${BOLD}KubeStack-AI CLI (ksa)${RESET} - Unified AI-powered middleware diagnosis for Kubernetes & baremetal.

${BOLD}Usage:${RESET}
  ksa [command] [flags]

${BOLD}Available Commands:${RESET}
  diagnose     Diagnose middleware (诊断中间件)
  optimize     Provide performance optimization suggestions (性能优化建议)
  fix          Apply safe fixes (应用安全修复)
  plugins      Manage diagnosis plugins (管理诊断插件)
  context      Show collected runtime context (显示采集的上下文信息)
  update       Update knowledge base (更新知识库)
  help         Show this help (显示帮助)

${BOLD}Flags:${RESET}
  -m, --middleware string   Specify middleware (e.g. redis, mysql, kafka)
  -n, --namespace string    Specify namespace (default: all)
  -o, --output string       Output format: table|json (default "table")
  -l, --language string     Output language: en|zh (default: en)
  -h, --help                Show help

EOF
sleep 2

# --- Output Format (Table / JSON) ---
output_format="table"
while getopts "o:" opt; do
    case ${opt} in
        o) output_format=$OPTARG ;;
        *) ;;
    esac
done

# --- JSON Output Formatter ---
output_json() {
    echo -e "{"
    echo -e "  \"status\": \"$1\","
    echo -e "  \"finding\": \"$2\","
    echo -e "  \"recommendations\": ["
    echo -e "    {"
    echo -e "      \"description\": \"$3\","
    echo -e "      \"command\": \"$4\""
    echo -e "    }"
    echo -e "  ]"
    echo -e "}"
}

# --- Table Output Formatter ---
output_table() {
    echo -e "${BOLD}${BLUE}Diagnosis Result for $1${RESET}"
    echo "+--------------------+----------------------------------------------+"
    echo "| ${BOLD}Status 状态${RESET}     | ${YELLOW}WARNING 警告${RESET}                                     |"
    echo "+--------------------+----------------------------------------------+"
    echo "| Memory Usage       | 85%                                          |"
    echo "| Replication Lag    | 0.2s                                         |"
    echo "| AOF Rewrite Freq   | High                                         |"
    echo "+--------------------+----------------------------------------------+"
    echo ""
    echo "${BOLD}Root Cause Analysis 根因分析:${RESET}"
    echo "- High memory usage due to excessive caching"
    echo "- Frequent AOF rewrites impacting latency"
    echo ""
    echo "${BOLD}Recommendations 建议:${RESET}"
    echo "1. Increase Redis maxmemory to 2Gi"
    echo "2. Reduce AOF rewrite frequency via 'auto-aof-rewrite-percentage'"
    echo ""
    echo "${BOLD}Suggested Fix Command 建议修复命令:${RESET}"
    echo "  ksa fix --middleware redis --action tune-memory --value 2Gi"
    echo ""
}

# --- Simulated command: Diagnose Redis ---
echo -e "${BOLD}${GREEN}>>> Task: ksa diagnose --middleware redis --namespace prod${RESET}"
progress "Collecting metrics from Redis in namespace 'prod' 收集中间件指标"
progress "Analyzing logs 分析日志"
progress "Running AI-powered root cause analysis 执行AI根因分析"

if [[ "$output_format" == "json" ]]; then
    output_json "warning" "High memory usage and frequent AOF rewrites" "Increase maxmemory and adjust AOF settings" "ksa fix --middleware redis --action tune-memory --value 2Gi"
else
    # Redis Metrics & Logs
    echo -e "${BOLD}Redis Metrics 监控指标${RESET}"
    echo "+---------------------+-----------------+"
    echo "| Metric              | Value           |"
    echo "+---------------------+-----------------+"
    echo "| Memory Usage        | 85%             |"
    echo "| Connected Clients   | 1200            |"
    echo "| Replication Lag     | 0.2s            |"
    echo "| AOF Rewrite Pending | 10s             |"
    echo "+---------------------+-----------------+"
    echo ""
    
    echo -e "${BOLD}Redis Logs 日志${RESET}"
    echo "+--------------------------+----------------------------------------+"
    echo "| Timestamp                | Log Message                            |"
    echo "+--------------------------+----------------------------------------+"
    echo "| [2025-08-09 12:01:23]    | ERROR: Memory usage is 85%, increase capacity |"
    echo "| [2025-08-09 12:00:10]    | WARNING: AOF rewrite pending for 10s    |"
    echo "| [2025-08-09 11:55:05]    | INFO: Replication lag: 0.2s           |"
    echo "+--------------------------+----------------------------------------+"
    echo ""

    echo "[Redis] Mocked diagnosis: High memory usage and frequent AOF rewrites"
    echo "Recommendation: Increase maxmemory and reduce AOF rewrite frequency"
    echo ""
fi

# --- Simulated command: Diagnose MinIO ---
echo -e "${BOLD}${GREEN}>>> Task: ksa diagnose -m minio${RESET}"
progress "Checking bucket policies 检查存储桶策略"
progress "Validating user permissions 验证用户权限"
if [[ "$output_format" == "json" ]]; then
    output_json "warning" "Conflicting bucket policies for 'logs-bucket'" "Review and unify bucket access rules" "ksa fix --middleware minio --action fix-policy"
else
    echo "[MinIO] Mocked diagnosis: Conflicting bucket policies for 'logs-bucket'"
    echo "Recommendation: Review and unify bucket access rules"
    echo ""
fi

# --- Simulated command: Diagnose MySQL ---
echo -e "${BOLD}${GREEN}>>> Task: ksa diagnose -m mysql${RESET}"
progress "Checking replication status 检查主从同步状态"
progress "Analyzing slow queries 分析慢查询日志"
if [[ "$output_format" == "json" ]]; then
    output_json "critical" "Replication delay 12s" "Optimize binlog settings and network latency" "ksa fix --middleware mysql --action optimize-binlog"
else
    echo "[MySQL] Mocked diagnosis: Replication delay 12s"
    echo "Recommendation: Optimize binlog settings, check network latency"
    echo ""
fi

# --- Simulated command: Diagnose Kafka ---
echo -e "${BOLD}${GREEN}>>> Task: ksa diagnose -m kafka${RESET}"
progress "Checking broker health 检查Broker健康"
progress "Validating ISR count and message lag 验证ISR数量与消息延迟"
if [[ "$output_format" == "json" ]]; then
    output_json "warning" "High message lag and ISR count dropping" "Scale Kafka brokers and check network health" "ksa fix --middleware kafka --action scale-brokers"
else
    echo "[Kafka] Mocked diagnosis: High message lag"
    echo "Recommendation: Scale Kafka brokers and check network health"
    echo ""
fi

# --- Simulated command: Diagnose ElasticSearch ---
echo -e "${BOLD}${GREEN}>>> Task: ksa diagnose -m elasticsearch${RESET}"
progress "Checking cluster health 检查集群健康"
progress "Validating shard allocation and performance 验证分片分配与性能"
if [[ "$output_format" == "json" ]]; then
    output_json "critical" "Cluster health is red" "Rebalance shards and check node resources" "ksa fix --middleware elasticsearch --action rebalance-shards"
else
    echo "[ElasticSearch] Mocked diagnosis: Cluster health is red"
    echo "Recommendation: Rebalance shards and check node resources"
    echo ""
fi

# --- Simulate RAG Knowledge Retrieval ---
echo -e "${BOLD}${GREEN}>>> Task: RAG knowledge retrieval for Redis AOF optimization${RESET}"
progress "Retrieving relevant knowledge from Redis best practices retrieval system 从Redis最佳实践系统检索相关知识"
if [[ "$output_format" == "json" ]]; then
    output_json "info" "Fetched Redis AOF tuning best practices" "Suggested AOF tuning: Set 'auto-aof-rewrite-percentage' to 10" "ksa fix --middleware redis --action optimize-aof --value 10"
else
    echo "[RAG] Fetched Redis AOF tuning best practices: Set 'auto-aof-rewrite-percentage' to 10"
    echo "Recommendation: Set 'auto-aof-rewrite-percentage' to 10"
    echo ""
fi

# --- Simulated command: Update Knowledge Base ---
echo -e "${BOLD}${GREEN}>>> Task: ksa update --knowledge-base redis --action add --doc 'Redis AOF rewrite tuning best practices'${RESET}"
progress "Fetching knowledge base update data 获取知识库更新"
progress "Adding new knowledge to Redis section 更新Redis知识库"
echo -e "${GREEN}✔ Knowledge base for Redis updated successfully. Redis知识库更新成功。${RESET}"

# --- Simulated command: Fix Command ---
echo -e "${BOLD}${GREEN}>>> Task: ksa fix --middleware redis --action tune-memory --value 2Gi${RESET}"
echo -e "${YELLOW}High-risk operation detected. Proceed? (y/n) 高风险操作，是否继续？${RESET}"
read confirm
if [[ "$confirm" =~ ^[Yy]$ ]]; then
    progress "Applying fix 应用修复"
    echo -e "${GREEN}✔ Redis memory tuned successfully. 修复完成。${RESET}"
else
    echo -e "${CYAN}ℹ Operation cancelled. 已取消操作。${RESET}"
fi

echo ""
echo -e "${BOLD}Demo complete. Thank you for exploring KubeStack-AI! 演示完成，感谢体验！${RESET}"
