variable "project_id" { type = string }
variable "project_region" { type = string }
variable "network_name" { type = string }
variable "subnetwork_name" { type = string }

variable "db_secrets" {
  type = object({
    db_host     = string
    db_name     = string
    db_password = string
    db_user     = string
  })
}

variable "monitoring_license_key_secret_id" { type = string }
variable "cloudrun_settings" { type = any }
variable "async_worker_settings" { type = any }
variable "api_server_settings" { type = any }
variable "monitoring_settings_apm" { type = any }
