resource "google_compute_address" "nat_gateway" {
  name         = var.bastion_settings.compute_address_name
  project      = var.project_id
  region       = var.project_region
  address_type = "EXTERNAL"
  network_tier = "PREMIUM"
}

resource "google_compute_router" "bastion_nat_router" {
  name    = "bastion-nat-router"
  region  = var.project_region
  network = var.network_id

  encrypted_interconnect_router = false
}

resource "google_compute_router_nat" "bastion_nat" {
  name                                = "bastion-nat"
  region                              = google_compute_router.bastion_nat_router.region
  router                              = google_compute_router.bastion_nat_router.name
  source_subnetwork_ip_ranges_to_nat  = "ALL_SUBNETWORKS_ALL_IP_RANGES"
  nat_ip_allocate_option              = "AUTO_ONLY"
  min_ports_per_vm                    = 64
  enable_endpoint_independent_mapping = false
  enable_dynamic_port_allocation      = false

  log_config {
    enable = false
    filter = "ALL"
  }
}

resource "google_compute_firewall" "iap_allow_ingress" {
  name        = "iap-allow-ingress"
  description = "IAPからの接続を許可する"

  network       = var.network_name
  direction     = "INGRESS"
  priority      = 1000
  source_ranges = ["35.235.240.0/20"]
  target_tags   = ["iap"]
  allow {
    protocol = "tcp"
    ports    = ["22", "5432"]
  }
}

resource "google_compute_instance" "bastion" {
  name         = "bastion"
  machine_type = var.bastion_settings.machine_type
  zone         = var.bastion_settings.zone

  tags = ["iap"]

  boot_disk {
    initialize_params {
      image = "debian-12-bookworm-v20240213"
      size  = 10
    }
  }

  network_interface {
    network    = var.network_name
    subnetwork = var.subnetwork_name
    stack_type = "IPV4_ONLY"
  }

  service_account {
    scopes = ["cloud-platform"]
  }

  scheduling {
    automatic_restart   = true
    on_host_maintenance = "MIGRATE"
    preemptible         = false
    provisioning_model  = "STANDARD"
  }

  enable_display      = false
  deletion_protection = false
}
