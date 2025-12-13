output "service_name" {
  value = google_cloud_run_service.async_worker.name
}

output "service_account_email" {
  value = google_service_account.async_worker_sa.email
}
