
terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "3.88.0"
    }
  }
}

provider "google" {
  user_project_override = true
  billing_project = "fabled-ray-104117"
}


resource "google_cloud_identity_group_membership" "cloud_identity_group_membership_basic" {
  group = "groups/02grqrue4gb58m7"

  preferred_member_key {
    id = "user1@esodemoapp2.com"
  }
  roles {
    name = "MEMBER"
  }
}


data "google_cloud_identity_group_memberships" "members" {
  group = "groups/02grqrue4gb58m7"
}

output "members" {
  value = data.google_cloud_identity_group_memberships.members.memberships
}