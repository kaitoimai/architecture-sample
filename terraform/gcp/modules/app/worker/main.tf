resource "google_cloud_run_service" "async_worker" {
  provider                   = google
  name                       = "async-worker"
  location                   = var.project_region
  autogenerate_revision_name = true

  metadata {
    annotations = {
      "run.googleapis.com/ingress" = "internal"
    }
  }

  template {
    spec {
      service_account_name = google_service_account.async_worker_sa.email
      containers {
        image = "asia-northeast1-docker.pkg.dev/${var.project_id}/docker/async-worker:latest"
        command = [
          "./server",
          "-mode=job"
        ]
        resources {
          limits = {
            "cpu" : var.cloudrun_settings.async_worker.cpu
            "memory" : var.cloudrun_settings.async_worker.memory
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
            "APP_GCP_CLOUD_RUN_SERVICE_ACCOUNT_EMAIL" = google_service_account.async_worker_sa.email
            "MONITORING_APP_NAME"                     = var.monitoring_settings_apm.apm.app_name
            "MONITORING_ENABLED"                      = var.monitoring_settings_apm.apm.enabled_string
            "APP_LOG_PUBSUB_PROJECT_ID"               = var.project_id
            "APP_LOG_PUBSUB_TOPIC_NAME"               = var.api_server_settings.topic
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
        "run.googleapis.com/startupProbeType" = "Custom"
      }
      annotations = {
        "run.googleapis.com/client-name"       = "terraform"
        "run.googleapis.com/vpc-access-egress" = "all-traffic"
        "run.googleapis.com/network-interfaces" = jsonencode(
          [
            {
              network    = var.network_name
              subnetwork = var.subnetwork_name
            },
          ]
        )
        "autoscaling.knative.dev/maxScale" = var.async_worker_settings.max_scale
        "autoscaling.knative.dev/minScale" = var.async_worker_settings.min_scale
      }
    }
  }

  lifecycle {
    ignore_changes = [
      metadata[0].annotations
    ]
  }
}

resource "google_service_account" "async_worker_sa" {
  account_id   = "async-worker-sa"
  display_name = "Service Account for Async Worker"
}

locals {
  async_worker_roles = [
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

resource "google_project_iam_member" "async_worker_iam" {
  for_each = toset(local.async_worker_roles)
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.async_worker_sa.email}"
}
