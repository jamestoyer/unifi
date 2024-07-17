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

data "unifi_device_switch" "example" {
  mac = "00:27:22:00:00:01"
}

import {
  id = data.unifi_device_switch.example.id
  to = unifi_device_switch.example
}

resource "unifi_device_switch" "example" {
  name = "Example Switch"
  mac  = data.unifi_device_switch.example.mac

  #   disabled = true
  #     snmp_contact = "a"
}

output "unifi_device_switch" {
  value = unifi_device_switch.example
}
