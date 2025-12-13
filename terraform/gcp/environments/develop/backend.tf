terraform {
  backend "gcs" {
    bucket = "terraform-state-develop"
    prefix = "terraform/state"
  }
}
