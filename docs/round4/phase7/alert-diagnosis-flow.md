# Monitoring, Alerting, and Diagnosis Integration

## Overview

This document describes the flow from an external alert (e.g., Prometheus) to the diagnosis engine and back to the user via notifications.

## Architecture

The system consists of the following components:

1.  **Webhook Handler**: Receives alerts from Alertmanager or Grafana.
2.  **Dispatcher**: Deduplicates alerts and routes them.
3.  **Correlator**: Aggregates related alerts (same instance, time window) into a single diagnosis request.
4.  **Diagnosis Manager**: Executes the diagnosis (Collection -> Analysis -> Reporting).
5.  **Feedback Processor**: Sends the diagnosis report to configured notification channels (DingTalk, Slack).

## Flow

1.  **Alert Ingestion**:
    - Prometheus/Grafana sends a webhook to `/api/v1/webhook/alertmanager` or `/grafana`.
    - `WebhookHandler` parses the payload and converts it to internal `AlertEvent`.

2.  **Dispatch & Correlation**:
    - `Dispatcher` receives the event.
    - It checks for duplicates (deduplication window).
    - Valid alerts are passed to `Correlator`.
    - `Correlator` groups alerts by instance.
    - After the correlation window (or immediately for Critical alerts), a `CorrelatedAlert` is produced.

3.  **Diagnosis Trigger**:
    - `Dispatcher` triggers `DiagnosisManager.DiagnoseFromAlert`.
    - A `DiagnosisRequest` is created targeting the alerted instance.
    - The diagnosis runs asynchronously.

4.  **Feedback**:
    - Upon completion, `Dispatcher` passes the `DiagnosisResult` to `FeedbackProcessor`.
    - `FeedbackProcessor` formats the message (Markdown).
    - It sends the notification to enabled channels (DingTalk, Slack, etc.) based on severity filters.

## Configuration

### Alert Rules (`configs/alert_rules.yaml`)

Defines how alerts map to diagnosis scopes and auto-fix strategies.

```yaml
alert_rules:
  - name: "RedisMemoryHigh"
    middleware: redis
    diagnosis_scope: ["memory"]
    auto_fix: true
```

### Notification Channels

Configured in `configs/alert_rules.yaml` (or main config).

```yaml
notification:
  channels:
    - type: dingtalk
      webhook_url: "..."
      secret: "..."
```

## Adding New Notifiers

Implement the `Notifier` interface in `internal/alert/notifier` and register it in `Manager`.
