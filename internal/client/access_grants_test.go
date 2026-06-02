package client

import (
	"reflect"
	"testing"
)

func TestAccessControlToGrants(t *testing.T) {
	ac := map[string]any{
		"read":  map[string]any{"group_ids": []string{"g1", "g2"}, "user_ids": []string{}},
		"write": map[string]any{"group_ids": []string{"g2"}, "user_ids": []string{}},
	}
	grants := accessControlToGrants(ac)
	if len(grants) != 3 {
		t.Fatalf("expected 3 grants, got %d: %+v", len(grants), grants)
	}

	want := map[string]bool{"group/g1/read": true, "group/g2/read": true, "group/g2/write": true}
	for _, g := range grants {
		key := g.PrincipalType + "/" + g.PrincipalID + "/" + g.Permission
		if !want[key] {
			t.Fatalf("unexpected grant %q in %+v", key, grants)
		}
	}
}

func TestAccessControlToGrantsNil(t *testing.T) {
	grants := accessControlToGrants(nil)
	if grants == nil || len(grants) != 0 {
		t.Fatalf("expected empty non-nil slice, got %+v", grants)
	}
}

func TestGrantsToAccessControl(t *testing.T) {
	grants := []accessGrant{
		{PrincipalType: "group", PrincipalID: "g1", Permission: "read"},
		{PrincipalType: "group", PrincipalID: "g2", Permission: "write"},
		{PrincipalType: "user", PrincipalID: "u1", Permission: "read"},
		{PrincipalType: "user", PrincipalID: "*", Permission: "read"}, // wildcard, skipped
	}
	ac := grantsToAccessControl(grants)
	read, ok := ac["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected read to be map[string]any, got %T", ac["read"])
	}
	write, ok := ac["write"].(map[string]any)
	if !ok {
		t.Fatalf("expected write to be map[string]any, got %T", ac["write"])
	}
	if !reflect.DeepEqual(read["group_ids"], []string{"g1"}) {
		t.Fatalf("read.group_ids = %+v", read["group_ids"])
	}
	if !reflect.DeepEqual(read["user_ids"], []string{"u1"}) {
		t.Fatalf("read.user_ids = %+v", read["user_ids"])
	}
	if !reflect.DeepEqual(write["group_ids"], []string{"g2"}) {
		t.Fatalf("write.group_ids = %+v", write["group_ids"])
	}
}

func TestGrantsToAccessControlEmpty(t *testing.T) {
	if ac := grantsToAccessControl(nil); ac != nil {
		t.Fatalf("expected nil, got %+v", ac)
	}
	onlyWildcard := []accessGrant{{PrincipalType: "user", PrincipalID: "*", Permission: "read"}}
	if ac := grantsToAccessControl(onlyWildcard); ac != nil {
		t.Fatalf("expected nil for wildcard-only, got %+v", ac)
	}
}
