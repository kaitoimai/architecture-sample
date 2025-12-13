# Primary database credentials
data "google_secret_manager_secret" "db_host" {
  secret_id = "DB_HOST"
}

data "google_secret_manager_secret" "db_name" {
  secret_id = "DB_NAME"
}

data "google_secret_manager_secret" "db_password" {
  secret_id = "DB_PASSWORD"
}

data "google_secret_manager_secret" "db_user" {
  secret_id = "DB_USER"
}

# Monitoring/APM service
data "google_secret_manager_secret" "monitoring_license_key" {
  secret_id = "MONITORING_LICENSE_KEY"
}
