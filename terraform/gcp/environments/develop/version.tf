terraform {
  required_version = "1.5.2"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.84"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "4.84"
    }
  }
}
