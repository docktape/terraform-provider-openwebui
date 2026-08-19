package provider

import (
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func TestAdminConfigFormModelRoundTrip(t *testing.T) {
	folderMax := int64(100)
	automationMax := int64(50)
	automationMin := int64(1000)
	memSysCtx := true
	overlayTitle := "Welcome"
	watermark := "Powered by Open WebUI"

	form := client.AdminConfigForm{
		ShowAdminDetails:                  true,
		AdminEmail:                        strPtr("admin@example.com"),
		WebUIURL:                          "https://webui.example.com",
		EnableSignup:                      false,
		EnableAPIKeys:                     true,
		EnableAPIKeysEndpointRestrictions: false,
		APIKeysAllowedEndpoints:           "/api/v1/chat/completions,/api/v1/models",
		DefaultUserRole:                   "user",
		DefaultGroupID:                    "default",
		JWTExpiresIn:                      "30d",
		EnableCommunitySharing:            true,
		EnableMessageRating:               false,
		EnableFolders:                     true,
		FolderMaxFileCount:                &folderMax,
		AutomationMaxCount:                &automationMax,
		AutomationMinInterval:             &automationMin,
		EnableAutomations:                 true,
		EnableChannels:                    false,
		EnableCalendar:                    true,
		EnableMemories:                    true,
		EnableMemorySystemContext:         &memSysCtx,
		EnableNotes:                       false,
		EnableUserWebhooks:                true,
		EnableUserStatus:                  false,
		PendingUserOverlayTitle:           &overlayTitle,
		PendingUserOverlayContent:         strPtr("Please wait for approval"),
		ResponseWatermark:                 &watermark,
	}
	model := adminConfigModelFromForm(form)

	// Verify key fields converted correctly
	if !model.ShowAdminDetails.ValueBool() {
		t.Fatal("expected show_admin_details=true")
	}
	if model.AdminEmail.ValueString() != "admin@example.com" {
		t.Fatalf("expected admin_email=admin@example.com, got %s", model.AdminEmail.ValueString())
	}
	if model.WebUIURL.ValueString() != "https://webui.example.com" {
		t.Fatalf("expected webui_url=https://webui.example.com, got %s", model.WebUIURL.ValueString())
	}
	if model.DefaultUserRole.ValueString() != "user" {
		t.Fatalf("expected default_user_role=user, got %s", model.DefaultUserRole.ValueString())
	}
	if model.FolderMaxFileCount.ValueInt64() != 100 {
		t.Fatalf("expected folder_max_file_count=100, got %d", model.FolderMaxFileCount.ValueInt64())
	}
	if !model.EnableMemorySystemContext.ValueBool() {
		t.Fatal("expected enable_memory_system_context=true")
	}
	if model.PendingUserOverlayTitle.ValueString() != "Welcome" {
		t.Fatalf("expected pending_user_overlay_title=Welcome, got %s", model.PendingUserOverlayTitle.ValueString())
	}

	// Test round-trip back to form
	roundTripped := adminConfigFormFromModel(model)
	if !roundTripped.ShowAdminDetails {
		t.Fatal("round-trip lost ShowAdminDetails")
	}
	if roundTripped.AdminEmail == nil || *roundTripped.AdminEmail != "admin@example.com" {
		t.Fatalf("round-trip lost AdminEmail: %v", roundTripped.AdminEmail)
	}
	if roundTripped.WebUIURL != "https://webui.example.com" {
		t.Fatalf("round-trip lost WebUIURL: %s", roundTripped.WebUIURL)
	}
	if roundTripped.DefaultUserRole != "user" {
		t.Fatalf("round-trip lost DefaultUserRole: %s", roundTripped.DefaultUserRole)
	}
	if roundTripped.FolderMaxFileCount == nil || *roundTripped.FolderMaxFileCount != 100 {
		t.Fatalf("round-trip lost FolderMaxFileCount: %v", roundTripped.FolderMaxFileCount)
	}
	if roundTripped.EnableMemorySystemContext == nil || !*roundTripped.EnableMemorySystemContext {
		t.Fatalf("round-trip lost EnableMemorySystemContext: %v", roundTripped.EnableMemorySystemContext)
	}
	if roundTripped.PendingUserOverlayTitle == nil || *roundTripped.PendingUserOverlayTitle != "Welcome" {
		t.Fatalf("round-trip lost PendingUserOverlayTitle: %v", roundTripped.PendingUserOverlayTitle)
	}
}
