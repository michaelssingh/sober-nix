terraform {
  required_version = ">= 1.0.0"
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}

# The Cloudflare provider will automatically look for the
# CLOUDFLARE_API_TOKEN environment variable in your shell.
provider "cloudflare" {}

variable "cloudflare_zone_id" {
  type        = string
  description = "The Zone ID for sober.fyi (can be found in Cloudflare overview sidebar)"
}

# git.sober.fyi CNAME pointing to Fly.io app
resource "cloudflare_record" "git" {
  zone_id = var.cloudflare_zone_id
  name    = "git"
  value   = "sober-bubo.fly.dev"
  type    = "CNAME"
  proxied = false # Must be false (DNS Only) to allow SSH connections (port 2222)
  ttl     = 1     # Automatic TTL
}
