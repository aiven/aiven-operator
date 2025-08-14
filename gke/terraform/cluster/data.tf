data "http" "my_ip" {
  # Calls an external service to get public IP address.
  # It's only used if the my_public_ip variable is not set.
  url = "https://ipv4.icanhazip.com"
}
