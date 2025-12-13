output "alert_policy_id" {
  value = google_monitoring_alert_policy.api_server_memory_high_utilization.id
}

output "monitoring_topic_name" {
  value = google_pubsub_topic.monitoring_logging.name
}
