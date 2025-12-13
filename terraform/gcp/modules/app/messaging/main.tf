# Pub/Sub topic for Cloud Build events
resource "google_pubsub_topic" "cloud_builds" {
  name = "cloud-builds"
}

# Allow GCP Pub/Sub service account to create tokens
resource "google_project_iam_member" "gcp_pub_sub_sa_token_creator_binding" {
  project = var.project_id
  role    = "roles/iam.serviceAccountTokenCreator"
  member  = "serviceAccount:service-${var.project_number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# Cloud Tasks queue for CSV processing background jobs
resource "google_cloud_tasks_queue" "csv_processing" {
  name     = "csv-processing"
  location = var.project_region

  rate_limits {
    max_concurrent_dispatches = 1
    max_dispatches_per_second = 1
  }
  retry_config {
    max_attempts       = 1
    max_backoff        = "3600s"
    max_doublings      = 16
    max_retry_duration = "10s"
    min_backoff        = "0.100s"
  }
  stackdriver_logging_config {
    sampling_ratio = 1
  }
}

# Eventarc trigger for CSV upload
# This trigger fires when a CSV file is uploaded to the CSV bucket
resource "google_eventarc_trigger" "csv_upload_to_api_server" {
  event_data_content_type = "application/json"
  labels                  = {}
  location                = var.project_region
  name                    = "trigger-csv-upload-to-api-server"
  project                 = var.project_id
  service_account         = var.api_server_sa_email

  destination {
    cloud_run_service {
      path    = "/v1/call/csv_import"
      region  = var.project_region
      service = var.api_server_service_name
    }
  }

  matching_criteria {
    attribute = "bucket"
    value     = var.csv_upload_bucket_name
  }
  matching_criteria {
    attribute = "type"
    value     = "google.cloud.storage.object.v1.finalized"
  }

  timeouts {}
}

# Allow Cloud Storage service account to publish to Pub/Sub
resource "google_project_iam_member" "storage_sa_role_binding" {
  project = var.project_id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${var.google_storage_sa}"
}
