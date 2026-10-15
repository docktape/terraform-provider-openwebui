variable "openai_api_key" {
  type      = string
  sensitive = true
}

resource "openwebui_images_config" "example" {
  enable_image_generation        = true
  enable_image_prompt_generation = true

  image_generation_engine = "openai"
  image_generation_model  = "gpt-image-1"
  image_size              = "1024x1024"

  images_openai_api_base_url    = "https://api.openai.com/v1"
  images_openai_api_key         = var.openai_api_key
  images_openai_api_params_json = jsonencode({ quality = "high" })

  enable_image_edit          = true
  image_edit_engine          = "openai"
  image_edit_model           = "gpt-image-1"
  images_edit_openai_api_key = var.openai_api_key
}
