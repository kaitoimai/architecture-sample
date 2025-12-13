output "service_name" {
  value = google_cloud_run_service.api_server.name
}

output "service_account_email" {
  value = google_service_account.api_server_sa.email
}
