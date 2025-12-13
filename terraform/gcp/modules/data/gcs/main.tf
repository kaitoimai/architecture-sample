# Terraform state backend bucket
resource "google_storage_bucket" "terraform_state" {
  name                     = "terraform-state-${var.environment}"
  location                 = var.project_region
  storage_class            = "STANDARD"
  public_access_prevention = "enforced"
  force_destroy            = false

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      num_newer_versions = 5
    }
    action {
      type = "Delete"
    }
  }
}

# Web client static hosting bucket
resource "google_storage_bucket" "web_client" {
  name                     = "web-client-${var.environment}"
  location                 = var.project_region
  storage_class            = "STANDARD"
  public_access_prevention = "inherited"
  force_destroy            = false

  website {
    main_page_suffix = "index.html"
    not_found_page   = "404.html"
  }
}

# CSV upload storage for API server
resource "google_storage_bucket" "csv_upload" {
  name                     = "csv-upload-${var.environment}"
  location                 = var.project_region
  storage_class            = "STANDARD"
  public_access_prevention = "enforced"
  force_destroy            = false

  uniform_bucket_level_access = true

  cors {
    origin = ["*"]
    method = [
      "GET",
      "PUT"
    ]
    max_age_seconds = 600
    response_header = [
      "Content-Type",
      "Access-Control-Allow-Origin",
      "x-goog-meta-import-task-id",
      "x-goog-meta-access-token"
    ]
  }
}

# Make web client bucket publicly accessible
resource "google_storage_bucket_iam_member" "web_client_viewer" {
  bucket = google_storage_bucket.web_client.name
  role   = "roles/storage.objectViewer"
  member = "allUsers"
}

# Service account for CI/CD (e.g., Github Actions)
resource "google_service_account" "web_client_deploy" {
  account_id   = "web-client-deploy"
  display_name = "Account for CI/CD to deploy web client"
}

resource "google_storage_bucket_iam_member" "web_client_deploy_owner" {
  bucket = google_storage_bucket.web_client.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.web_client_deploy.email}"
}

resource "google_project_iam_member" "web_client_deploy_compute_admin" {
  project = var.project_id
  role    = "roles/compute.admin"
  member  = "serviceAccount:${google_service_account.web_client_deploy.email}"
}
