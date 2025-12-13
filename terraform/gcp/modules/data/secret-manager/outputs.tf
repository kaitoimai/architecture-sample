output "db_host_secret_id" {
  value = data.google_secret_manager_secret.db_host.secret_id
}

output "db_name_secret_id" {
  value = data.google_secret_manager_secret.db_name.secret_id
}

output "db_password_secret_id" {
  value = data.google_secret_manager_secret.db_password.secret_id
}

output "db_user_secret_id" {
  value = data.google_secret_manager_secret.db_user.secret_id
}

output "monitoring_license_key_secret_id" {
  value = data.google_secret_manager_secret.monitoring_license_key.secret_id
}
