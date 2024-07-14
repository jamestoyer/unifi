terraform {
  required_providers {
    unifi = {
      source = "jamestoyer/unifi"
    }
  }
}

provider "unifi" {
  insecure = true
  url      = "https://127.0.0.1:8443"
  username = "admin"
  password = "admin"
}

data "unifi_device" "example" {
  mac = "dc:9f:db:00:00:01"
}

output "example" {
  value = data.unifi_device.example
}