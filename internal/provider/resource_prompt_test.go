package provider

import (
	"testing"
)

func TestPromptResourceModel_HasNewFields(t *testing.T) {
	m := promptResourceModel{}
	_ = m.IsActive
	_ = m.Tags
	_ = m.DataJSON
	_ = m.MetaJSON
}
