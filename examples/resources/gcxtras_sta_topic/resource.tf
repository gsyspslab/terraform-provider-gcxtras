resource "gcxtras_sta_topic" "example" {
  name         = "Customer Complaint"
  description  = "Detects when customers express complaints"
  strictness   = "72"
  participants = "External"
  dialect      = "en-GB"
  tags         = ["complaints", "cx"]
}
