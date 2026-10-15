variable "openai_api_key" {
  type      = string
  sensitive = true
}

resource "openwebui_audio_config" "example" {
  tts = {
    engine              = "openai"
    openai_api_base_url = "https://api.openai.com/v1"
    openai_api_key      = var.openai_api_key
    openai_params_json  = jsonencode({ speed = 1.1 })
    model               = "tts-1"
    voice               = "nova"
    split_on            = "punctuation"
  }

  stt = {
    engine                    = "openai"
    openai_api_base_url       = "https://api.openai.com/v1"
    openai_api_key            = var.openai_api_key
    openai_api_request_format = "multipart"
    model                     = "whisper-1"
    supported_content_types   = ["audio/*"]
    allowed_extensions        = ["mp3", "wav", "m4a", "webm", "ogg"]
  }
}
