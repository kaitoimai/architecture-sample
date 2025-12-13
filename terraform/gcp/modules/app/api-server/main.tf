resource "google_cloud_run_service" "api_server" {
  provider                   = google
  name                       = "api-server"
  location                   = var.project_region
  autogenerate_revision_name = true

  metadata {
    annotations = {
      "run.googleapis.com/ingress" = "internal-and-cloud-load-balancing"
    }
  }

  template {
    spec {
      service_account_name = google_service_account.api_server_sa.email
      timeout_seconds      = "600"
      containers {
        image = "asia-northeast1-docker.pkg.dev/${var.project_id}/docker/api-server:latest"
        resources {
          limits = {
            "cpu" : var.api_server_settings.cpu
            "memory" : var.api_server_settings.memory
          }
        }

        # Secret Managerから参照する環境変数
        dynamic "env" {
          for_each = {
            "DB_HOST"                = var.db_secrets.db_host
            "DB_NAME"                = var.db_secrets.db_name
            "DB_PASSWORD"            = var.db_secrets.db_password
            "DB_USER"                = var.db_secrets.db_user
            "MONITORING_LICENSE_KEY" = var.monitoring_license_key_secret_id
          }
          content {
            name = env.key
            value_from {
              secret_key_ref {
                name = env.value
                key  = "latest"
              }
            }
          }
        }

        # ハードコーディングされた環境変数
        dynamic "env" {
          for_each = {
            "APP_GCP_PROJECT_ID"                      = var.project_id
            "APP_GCP_LOCATION_ID"                     = var.project_region
            "APP_GCP_CLOUD_RUN_SERVICE_ACCOUNT_EMAIL" = google_service_account.api_server_sa.email
            "MONITORING_APP_NAME"                     = var.monitoring_settings_apm.apm.app_name
            "MONITORING_ENABLED"                      = var.monitoring_settings_apm.apm.enabled_string
            "APP_LOG_PUBSUB_PROJECT_ID"               = var.project_id
            "APP_LOG_PUBSUB_TOPIC_NAME"               = var.api_server_settings.topic
            "CLOUD_TASKS_QUEUE_ID"                    = "csv-processing"
            "CLOUD_TASKS_JOB_URL"                     = var.api_server_settings.cloud_tasks_data_import_job_url
          }
          content {
            name  = env.key
            value = env.value
          }
        }
      }
    }

    metadata {
      labels = {
        "run.googleapis.com/startupProbeType" = "Default"
      }
      annotations = {
        "run.googleapis.com/client-name"       = "terraform"
        "run.googleapis.com/vpc-access-egress" = "private-ranges-only"
        "run.googleapis.com/network-interfaces" = jsonencode(
          [
            {
              network    = var.network_name
              subnetwork = var.subnetwork_name
            },
          ]
        )
        "autoscaling.knative.dev/maxScale" = var.api_server_settings.max_scale
        "autoscaling.knative.dev/minScale" = var.api_server_settings.min_scale
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }

  lifecycle {
    ignore_changes = [
      metadata[0].annotations
    ]
  }
}

resource "google_service_account" "api_server_sa" {
  account_id   = "api-server-sa"
  display_name = "Account for API Server"
}

locals {
  api_server_roles = [
    "roles/secretmanager.secretAccessor",
    "roles/alloydb.client",
    "roles/pubsub.publisher",
    "roles/run.invoker",
    "roles/cloudtasks.admin",
    "roles/iam.serviceAccountTokenCreator",
    "roles/iam.serviceAccountUser",
    "roles/storage.admin",
    "roles/eventarc.eventReceiver"
  ]
}

resource "google_project_iam_member" "api_server_iam" {
  for_each = toset(local.api_server_roles)
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.api_server_sa.email}"
}

# Allow Load Balancer to invoke Cloud Run service
resource "google_cloud_run_service_iam_member" "api_server_public_invoker" {
  service  = google_cloud_run_service.api_server.name
  location = google_cloud_run_service.api_server.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
