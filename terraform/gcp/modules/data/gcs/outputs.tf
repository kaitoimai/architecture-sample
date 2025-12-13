output "web_client_bucket_name" {
  value = google_storage_bucket.web_client.name
}

output "csv_upload_bucket_name" {
  value = google_storage_bucket.csv_upload.name
}

output "web_client_deploy_sa_email" {
  value = google_service_account.web_client_deploy.email
}
