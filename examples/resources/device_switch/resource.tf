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
  name = "Example Switch Updates"
  mac  = data.unifi_device_switch.example.mac

  management_network_id = "669c0336329aae15c4b318f2"

  static_ip_settings = {
    ip            = "10.2.3.4"
    gateway       = "10.2.0.1"
    netmask       = "255.255.255.0"
    preferred_dns = "1.2.3.4"
  }

    port_overrides = {
      "39" = {
        full_duplex = true
        link_speed = "1000"
        name = "Port 39"
        native_network_id = "669c0336329aae15c4b318f2"
        operation = "switch"
      }
      "40" = {
        native_network_id = "669c0336329aae15c4b318f2"
        poe_mode = "off"
        name = "40"
      }
      "41"= {
        native_network_id = "669c0336329aae15c4b318f2"
        name = "Party Port"
      }
#       "42" = {
#         port_profile_id = "669c1ef8329aae15c4b3f791"
#         name = "Port 42"
#       }
#       "44" = {
#         native_network_id = ""
#       }
#       "45" ={
#         native_network_id = "669c0336329aae15c4b318f2"
#         name = "Port 45"
#       }
#       "46" = {
#         disabled = true
#         name = "Port 46"
#       }
    }
}
