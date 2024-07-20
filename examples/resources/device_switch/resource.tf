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

  management_network_id = "66994c4598129b0ccf324323"

  static_ip_settings = {
    ip            = "10.2.3.4"
    gateway       = "10.2.sdfds0.1"
    netmask       = "255.255.255.0"
    preferred_dns = "1.2.3.4"
  }

  #   port_overrides = {
  #     "39" = {
  #       full_duplex = true
  #       link_speed = "100"
  #       operation = "switch"
  #     }
  #     "40" = {
  #       poe_mode = "off"
  #     }
  #   }
}
