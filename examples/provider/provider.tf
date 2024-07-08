terraform {
  required_providers {
    unifi = {
      source = "jamestoyer/unifi"
    }
  }
}

provider "unifi" {
  allow_insecure = true
  url            = "https://127.0.0.1:8443"
  username       = "admin"
  password       = "admin"
}
