resource "gcxtras_sta_program" "example" {
  name        = "My_Analytics_Program"
  description = "A speech and text analytics program"
  published   = true
  tags        = ["terraform", "managed"]
  topic_ids = [
    data.gcxtras_sta_topic.knowledge_gap.id,
    gcxtras_sta_topic.custom_topic.id,
  ]
}
