package provider

import (
	"testing"
)

func TestBuildAccessControl_BothEmpty(t *testing.T) {
	result := buildAccessControl(nil, nil)
	if result != nil {
		t.Fatalf("expected nil when both slices empty, got %+v", result)
	}
}

func TestBuildAccessControl_ReadOnly(t *testing.T) {
	result := buildAccessControl([]string{"g1"}, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	read, ok := result["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected read section, got %T", result["read"])
	}
	ids, ok := read["group_ids"].([]string)
	if !ok {
		t.Fatalf("expected []string group_ids, got %T", read["group_ids"])
	}
	if len(ids) != 1 || ids[0] != "g1" {
		t.Fatalf("expected [g1], got %v", ids)
	}
	if _, hasWrite := result["write"]; hasWrite {
		t.Fatal("expected no write section for read-only access")
	}
}

func TestBuildAccessControl_ReadAndWrite(t *testing.T) {
	result := buildAccessControl([]string{"g1"}, []string{"g2"})
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Read should contain both g1 (read-only) and g2 (write implies read)
	read, ok := result["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected read section, got %T", result["read"])
	}
	readIDs, ok := read["group_ids"].([]string)
	if !ok {
		t.Fatalf("expected []string group_ids, got %T", read["group_ids"])
	}
	readSet := make(map[string]bool, len(readIDs))
	for _, id := range readIDs {
		readSet[id] = true
	}
	if !readSet["g1"] || !readSet["g2"] {
		t.Fatalf("expected read group_ids to include g1 and g2, got %v", readIDs)
	}

	// Write should contain only g2
	write, ok := result["write"].(map[string]any)
	if !ok {
		t.Fatalf("expected write section, got %T", result["write"])
	}
	writeIDs, ok := write["group_ids"].([]string)
	if !ok {
		t.Fatalf("expected []string group_ids, got %T", write["group_ids"])
	}
	if len(writeIDs) != 1 || writeIDs[0] != "g2" {
		t.Fatalf("expected write group_ids=[g2], got %v", writeIDs)
	}
}

func TestBuildAccessControl_WriteOnlyDeduplicatesRead(t *testing.T) {
	// g1 appears in both read and write — the merged read slice must not duplicate it
	result := buildAccessControl([]string{"g1"}, []string{"g1"})
	read, ok := result["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected read section, got %T", result["read"])
	}
	readIDs, ok := read["group_ids"].([]string)
	if !ok {
		t.Fatalf("expected []string group_ids, got %T", read["group_ids"])
	}
	if len(readIDs) != 1 {
		t.Fatalf("expected exactly 1 read group (deduplicated), got %v", readIDs)
	}
}

func TestExtractGroupIDsFromAccessControl_Nil(t *testing.T) {
	ids := extractGroupIDsFromAccessControl(nil, "read")
	if ids != nil {
		t.Fatalf("expected nil for nil access, got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_MissingSection(t *testing.T) {
	ids := extractGroupIDsFromAccessControl(map[string]any{}, "read")
	if ids != nil {
		t.Fatalf("expected nil for missing section, got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_SliceAny(t *testing.T) {
	access := map[string]any{
		"read": map[string]any{
			"group_ids": []any{"g1", "g2", ""},
			"user_ids":  []any{},
		},
	}
	ids := extractGroupIDsFromAccessControl(access, "read")
	// empty string is filtered out
	if len(ids) != 2 || ids[0] != "g1" || ids[1] != "g2" {
		t.Fatalf("expected [g1 g2], got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_SliceString(t *testing.T) {
	access := map[string]any{
		"write": map[string]any{
			"group_ids": []string{"g3"},
			"user_ids":  []string{},
		},
	}
	ids := extractGroupIDsFromAccessControl(access, "write")
	if len(ids) != 1 || ids[0] != "g3" {
		t.Fatalf("expected [g3], got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_SectionNotMap(t *testing.T) {
	access := map[string]any{"read": "unexpected-string"}
	ids := extractGroupIDsFromAccessControl(access, "read")
	if ids != nil {
		t.Fatalf("expected nil for non-map section, got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_MissingGroupIDs(t *testing.T) {
	access := map[string]any{
		"read": map[string]any{"user_ids": []any{"u1"}},
	}
	ids := extractGroupIDsFromAccessControl(access, "read")
	if ids != nil {
		t.Fatalf("expected nil for missing group_ids, got %v", ids)
	}
}
