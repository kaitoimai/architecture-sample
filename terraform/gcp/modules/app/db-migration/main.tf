resource "google_cloud_run_v2_job" "db_migration" {
  name     = "db-migration"
  location = var.project_region

  template {
    template {
      containers {
        image = "asia-northeast1-docker.pkg.dev/${var.project_id}/docker/db-migration:latest"

        # Secret Managerから参照する環境変数
        dynamic "env" {
          for_each = {
            "DB_HOST"     = var.db_secrets.db_host
            "DB_NAME"     = var.db_secrets.db_name
            "DB_PASSWORD" = var.db_secrets.db_password
            "DB_USER"     = var.db_secrets.db_user
          }
          content {
            name = env.key
            value_source {
              secret_key_ref {
                secret  = env.value
                version = "latest"
              }
            }
          }
        }

        env {
          name  = "DB_PORT"
          value = "5432"
        }

        resources {
          limits = {
            "cpu" : var.cloudrun_settings.db_migration.cpu
            "memory" : var.cloudrun_settings.db_migration.memory
          }
        }
      }

      execution_environment = "EXECUTION_ENVIRONMENT_GEN2"
      max_retries           = var.cloudrun_settings.db_migration.max_retries
      timeout               = var.cloudrun_settings.db_migration.timeout
      service_account       = google_service_account.db_migration_sa.email

      vpc_access {
        egress = "ALL_TRAFFIC"
        network_interfaces {
          network    = var.network_name
          subnetwork = var.subnetwork_name
        }
      }
    }
  }
}

resource "google_service_account" "db_migration_sa" {
  account_id   = "db-migration-sa"
  display_name = "Service Account for Database Migration Job"
}

locals {
  db_migration_roles = [
    "roles/secretmanager.secretAccessor",
    "roles/alloydb.client"
  ]
}

resource "google_project_iam_member" "db_migration_iam" {
  for_each = toset(local.db_migration_roles)
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.db_migration_sa.email}"
}
