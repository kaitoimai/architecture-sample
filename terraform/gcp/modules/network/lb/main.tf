# External IP Address for Load Balancer
resource "google_compute_global_address" "web_client_lb" {
  name         = "web-client-lb"
  address_type = "EXTERNAL"
}

# HTTPS Forwarding rule (port 443)
resource "google_compute_global_forwarding_rule" "web_client_https_forwarding" {
  name                  = "web-client-https-forwarding"
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "443"
  target                = google_compute_target_https_proxy.web_client_proxy.id
  ip_address            = google_compute_global_address.web_client_lb.address
}

# HTTP Forwarding rule for HTTPS redirect (port 80)
resource "google_compute_global_forwarding_rule" "web_client_http_redirect_forwarding" {
  name                  = "web-client-http-redirect-forwarding"
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "80"
  target                = google_compute_target_http_proxy.https_redirect_proxy.id
  ip_address            = google_compute_global_address.web_client_lb.address
}

# Target HTTPS proxy
resource "google_compute_target_https_proxy" "web_client_proxy" {
  name             = "web-client-proxy"
  url_map          = google_compute_url_map.web_client_map.id
  ssl_certificates = [google_compute_managed_ssl_certificate.web_client_ssl.id]
  ssl_policy       = google_compute_ssl_policy.app_ssl_policy.name
}

# Target HTTP proxy for redirect
resource "google_compute_target_http_proxy" "https_redirect_proxy" {
  name    = "https-redirect-proxy"
  url_map = google_compute_url_map.https_redirect_url_map.id
}

# Managed SSL certificate
resource "google_compute_managed_ssl_certificate" "web_client_ssl" {
  name = "web-client-ssl"
  managed {
    domains = [var.web_client_domain]
  }
}

# SSL policy
resource "google_compute_ssl_policy" "app_ssl_policy" {
  name            = "app-ssl-policy"
  profile         = "MODERN"
  min_tls_version = "TLS_1_2"
}

# URL map
resource "google_compute_url_map" "web_client_map" {
  name            = "web-client-map"
  default_service = google_compute_backend_bucket.web_client_bucket.id

  host_rule {
    hosts        = [var.web_client_domain]
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = google_compute_backend_bucket.web_client_bucket.id

    path_rule {
      paths   = ["/api/*"]
      service = google_compute_backend_service.api_server_backend.id
    }
  }
}

# URL map for HTTP to HTTPS redirect
resource "google_compute_url_map" "https_redirect_url_map" {
  name = "https-redirect-url-map"

  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }
}

# Backend bucket (connected to Cloud Storage)
resource "google_compute_backend_bucket" "web_client_bucket" {
  name        = "web-client-bucket"
  bucket_name = var.web_client_bucket_name
  enable_cdn  = true
}

# Serverless NEG for API Server Cloud Run
resource "google_compute_region_network_endpoint_group" "api_server_neg" {
  name                  = "api-server-neg"
  network_endpoint_type = "SERVERLESS"
  region                = var.project_region

  cloud_run {
    service = var.api_server_service_name
  }
}

# Cloud Armor Security Policy
resource "google_compute_security_policy" "cloud_armor_policy" {
  name        = "cloud-armor-policy"
  description = "Cloud Armor security policy with geo-restriction, IP allowlist, and SQL injection protection"

  # Rule 1: Allow traffic from Japan only
  rule {
    action   = "allow"
    priority = 10
    match {
      expr {
        expression = "origin.region_code == 'JP'"
      }
    }
    description = "Allow traffic from Japan only"
  }

  # Rule 2: Allow traffic from specific IP addresses (company office IPs)
  rule {
    action   = "allow"
    priority = 20
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = var.allowed_ip_ranges
      }
    }
    description = "Allow traffic from specific IP addresses"
  }

  # Rule 3: Block SQL injection attempts
  rule {
    action   = "deny(403)"
    priority = 30
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sqli-v33-stable')"
      }
    }
    description = "Block SQL injection attempts"
  }

  # Default rule: Deny all other traffic
  rule {
    action   = "deny(403)"
    priority = 2147483647
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "Default deny all"
  }

  # Adaptive Protection (auto-learning DDoS protection)
  adaptive_protection_config {
    layer_7_ddos_defense_config {
      enable = true
    }
  }
}

# Backend Service for API Server
resource "google_compute_backend_service" "api_server_backend" {
  name                  = "api-server-backend"
  protocol              = "HTTPS"
  port_name             = "http"
  timeout_sec           = 30
  enable_cdn            = false
  load_balancing_scheme = "EXTERNAL_MANAGED"
  security_policy       = google_compute_security_policy.cloud_armor_policy.id

  backend {
    group = google_compute_region_network_endpoint_group.api_server_neg.id
  }

  log_config {
    enable      = true
    sample_rate = 1.0
  }
}
