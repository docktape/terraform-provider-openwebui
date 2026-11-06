package client

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

// TestKnowledgeFormWireKeys pins the request body of knowledge create/update to
// the three keys Open WebUI's KnowledgeForm accepts (v0.11.0 and v0.12.0:
// name, description, access_grants). Every exported field of the Go struct is
// populated through reflection, so a field that is added back (for example
// data or meta, which the server ignores) shows up as an extra key.
func TestKnowledgeFormWireKeys(t *testing.T) {
	form := KnowledgeForm{
		AccessControl: map[string]any{
			"read":  map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{}, "user_ids": []string{}},
		},
	}
	v := reflect.ValueOf(&form).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		switch f.Kind() {
		case reflect.String:
			f.SetString("populated")
		case reflect.Map:
			if v.Type().Field(i).Name == "AccessControl" {
				continue
			}
			f.Set(reflect.MakeMapWithSize(f.Type(), 1))
			f.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf(1))
		}
	}

	body, err := json.Marshal(form)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var sent map[string]any
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	keys := make([]string, 0, len(sent))
	for k := range sent {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := []string{"access_grants", "description", "name"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("wire keys = %v, want %v (body %s)", keys, want, body)
	}
	if sent["name"] != "populated" || sent["description"] != "populated" {
		t.Fatalf("unexpected name/description: %+v", sent)
	}
	if grants, ok := sent["access_grants"].([]any); !ok || len(grants) != 1 {
		t.Fatalf("expected one access grant, got %+v", sent["access_grants"])
	}
}
