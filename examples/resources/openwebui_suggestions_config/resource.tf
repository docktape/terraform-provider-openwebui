resource "openwebui_suggestions_config" "example" {
  suggestions = [
    {
      title   = ["Explain this", "in simple terms"]
      content = "Explain the following concept in simple terms that a non-expert could understand:"
    },
    {
      title   = ["Summarize"]
      content = "Summarize the following in 3 bullet points:"
    },
  ]
}
