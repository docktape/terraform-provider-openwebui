package client

// accessGrant is the Open WebUI wire representation of a single access grant.
type accessGrant struct {
	ID            string `json:"id,omitempty"`
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	Permission    string `json:"permission"`
}

// accessControlToGrants converts the provider's nested access_control map
// ({"read"|"write": {"group_ids": [...], "user_ids": [...]}}) into the flat
// access_grants list the Open WebUI API (v0.9.0+) expects. A nil map yields an
// empty list (owner-only / private).
func accessControlToGrants(ac map[string]any) []accessGrant {
	grants := []accessGrant{}
	if ac == nil {
		return grants
	}
	for _, permission := range []string{"read", "write"} {
		section, ok := ac[permission].(map[string]any)
		if !ok {
			continue
		}
		for _, id := range anyToStrings(section["group_ids"]) {
			grants = append(grants, accessGrant{PrincipalType: "group", PrincipalID: id, Permission: permission})
		}
		for _, id := range anyToStrings(section["user_ids"]) {
			grants = append(grants, accessGrant{PrincipalType: "user", PrincipalID: id, Permission: permission})
		}
	}
	return grants
}

// grantsToAccessControl converts an access_grants list back into the provider's
// nested access_control map. Wildcard ("*") principals are skipped because the
// provider models access strictly by concrete group/user IDs. Returns nil when
// there are no concrete grants.
func grantsToAccessControl(grants []accessGrant) map[string]any {
	read := map[string]any{"group_ids": []string{}, "user_ids": []string{}}
	write := map[string]any{"group_ids": []string{}, "user_ids": []string{}}
	sections := map[string]map[string]any{"read": read, "write": write}

	found := false
	for _, g := range grants {
		if g.PrincipalID == "" || g.PrincipalID == "*" {
			continue
		}
		section, ok := sections[g.Permission]
		if !ok {
			continue
		}
		var key string
		switch g.PrincipalType {
		case "group":
			key = "group_ids"
		case "user":
			key = "user_ids"
		default:
			continue
		}
		section[key] = append(section[key].([]string), g.PrincipalID)
		found = true
	}
	if !found {
		return nil
	}
	return map[string]any{"read": read, "write": write}
}

func anyToStrings(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
