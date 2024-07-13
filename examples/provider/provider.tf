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

resource "unifi_user" "test" {
  mac  = "01:23:45:67:89:ab"
  name = "some client"
  fixed_ip = "192.168.1.30"
}