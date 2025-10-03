resource "doppler_service_account" "ci" {
  name = "ci"
  workplace_role = "collaborator"
}

resource "doppler_service_account_identity" "github_oidc" {
  service_account_slug = doppler_service_account.ci.slug
  name = "GitHub Actions OIDC"
  ttl_seconds = 600
  config_oidc {
    discovery_url = "https://token.actions.githubusercontent.com"
    claims_type = "wildcard"
    claims {
      key = "aud"
      values = ["https://github.com/DopplerHQ"]
    }
    claims {
      key = "sub"
      values = [
        "repo:DopplerHQ/terraform-provider-doppler:pull_request",
        "repo:DopplerHQ/terraform-provider-doppler:ref:refs/heads/feature*",
      ]
    }
  }
}