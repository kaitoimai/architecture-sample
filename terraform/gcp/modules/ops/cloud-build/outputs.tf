output "api_server_trigger_id" {
  value = google_cloudbuild_trigger.api_server_deploy.id
}

output "web_client_trigger_id" {
  value = google_cloudbuild_trigger.web_client_deploy.id
}
