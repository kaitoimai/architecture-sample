# Alert policy for API server high memory utilization
resource "google_monitoring_alert_policy" "api_server_memory_high_utilization" {
  display_name          = "Cloud Run (api-server) memory high utilization"
  combiner              = "OR"
  enabled               = var.monitoring_settings.enabled_alert
  notification_channels = var.monitoring_settings.notification_channels

  conditions {
    display_name = "Memory utilization (99th percentile)"
    condition_threshold {
      filter          = <<-EOF
        metric.type="run.googleapis.com/container/memory/utilizations" AND
        resource.type="cloud_run_revision" AND
        resource.labels.service_name="api-server"
      EOF
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_PERCENTILE_99"
      }
      trigger {
        count   = 1
        percent = 0
      }
    }
  }

  documentation {
    content   = <<-EOF
      Cloud Run (api-server) memory utilization has exceeded 80%.
      Please check the incidents page:
      https://console.cloud.google.com/monitoring/alerting/incidents?project=${var.project_id}
    EOF
    mime_type = "text/markdown"
  }
}

# External monitoring service (e.g., New Relic, Datadog) logging integration
# This example shows how to forward GCP logs to an external monitoring service

# Log sink to forward GCP logs to Pub/Sub
resource "google_logging_project_sink" "monitoring_log_sink" {
  name                   = "monitoring_log_sink"
  destination            = "pubsub.googleapis.com/projects/${var.project_id}/topics/${google_pubsub_topic.monitoring_logging.name}"
  unique_writer_identity = true
  # TODO: Add filter to send only specific logs
  # filter = "resource.type=\"cloud_run_revision\""
}

# Pub/Sub topic for monitoring service
resource "google_pubsub_topic" "monitoring_logging" {
  name = "monitoring_logging_${var.environment}"
}

# Pub/Sub subscription that pushes logs to external monitoring service
resource "google_pubsub_subscription" "monitoring_logging_subscription" {
  name  = "monitoring_logging_subscription_${var.environment}"
  topic = google_pubsub_topic.monitoring_logging.name

  push_config {
    attributes = {}
    # Replace with your actual monitoring service endpoint and API key
    push_endpoint = "https://log-api.monitoring-service.example/log/v1?Api-Key=YOUR_API_KEY&format=gcp"
  }
}

# Grant viewer access to monitoring service account
resource "google_project_iam_member" "monitoring_viewer_access" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "serviceAccount:${var.monitoring_service_account}"
}

# Grant service usage access to monitoring service account
resource "google_project_iam_member" "monitoring_service_usage_access" {
  project = var.project_id
  role    = "roles/serviceusage.serviceUsageConsumer"
  member  = "serviceAccount:${var.monitoring_service_account}"
}
