# Reminder: Terraform resource labels cannot begin with numbers.

### Logscale Groups
  resource "logscale_group" "test" {
    display_name = "test"
  }


# Add users to group
#resource "logscale_group_membership" "members" {
#  group_id = logscale_group.test.id
#  users = [
#    "example.user.one",
#    "example.user.two"
#  ]
#}


