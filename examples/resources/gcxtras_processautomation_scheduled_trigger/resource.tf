resource "gcxtras_processautomation_scheduled_trigger" "example" {
  name        = "Daily Report Generator"
  description = "Runs the daily report workflow every weekday at 9:00 AM"
  enabled     = true

  target {
    id   = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    type = "Workflow"
  }

  schedule {
    minutes       = [0]
    hours         = [9]
    days_of_week  = [2, 3, 4, 5, 6]
    timezone      = "Europe/London"
  }
}
