  terraform {
    required_providers {
      logscale = {
        source  = "local/logscale"
        version = "0.3.0"
      }
    }
  }

# A PAT may be needed for actions that organization tokens cannot perform, such as creating ingest tokens.
  provider "logscale" {
    api_url   = "https://your-tenant.logscale.region.crowdstrike.com/graphql"
    api_token = "replace-with-token"
  }
