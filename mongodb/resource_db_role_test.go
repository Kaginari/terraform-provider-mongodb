package mongodb

import (
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

// Privilege actions must be a TypeSet, not TypeList: MongoDB doesn't guarantee returning
// them in the order they were granted, and their order carries no meaning. A TypeList here
// made a reordered-but-identical actions list hash to a different privilege set element,
// so Terraform showed a full remove+add of the whole privilege block on every plan - #38.
func TestPrivilegeActionsIsTypeSet(t *testing.T) {
	privilegeElem := resourceDatabaseRole().Schema["privilege"].Elem.(*schema.Resource)
	if privilegeElem.Schema["actions"].Type != schema.TypeSet {
		t.Fatalf(`privilege.actions Type = %v, want schema.TypeSet`, privilegeElem.Schema["actions"].Type)
	}
}

// Reproduces the exact failure a TypeSet-typed nested attribute causes for
// mapstructure.Decode if left unnormalized: Get() on a TypeSet returns each element's
// "actions" as a *schema.Set (a struct), not a []interface{}, so mapstructure.Decode into
// []string fails with "source data must be an array or slice, got struct". This must be
// caught by normalizePrivilegeSet before decoding.
func TestNormalizePrivilegeSetDecodesCorrectly(t *testing.T) {
	raw := map[string]interface{}{
		"name": "test_role",
		"privilege": []interface{}{
			map[string]interface{}{
				"db":         "admin",
				"collection": "",
				"actions":    []interface{}{"find", "insert", "remove", "update"},
			},
		},
	}
	d := schema.TestResourceDataRaw(t, resourceDatabaseRole().Schema, raw)

	privilegeRaw := d.Get("privilege").(*schema.Set).List()
	normalized := normalizePrivilegeSet(privilegeRaw)

	var privileges []PrivilegeDto
	if err := mapstructure.Decode(normalized, &privileges); err != nil {
		t.Fatalf("mapstructure.Decode failed on normalized privilege set: %s", err)
	}
	if len(privileges) != 1 {
		t.Fatalf("got %d privileges, want 1", len(privileges))
	}

	got := append([]string{}, privileges[0].Actions...)
	sort.Strings(got)
	want := []string{"find", "insert", "remove", "update"}
	if len(got) != len(want) {
		t.Fatalf("actions = %v, want %v (same content)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("actions = %v, want %v (same content)", got, want)
		}
	}
	if privileges[0].Db != "admin" {
		t.Fatalf("Db = %q, want admin", privileges[0].Db)
	}
}

// normalizePrivilegeSet must not choke on, and must pass through unchanged, a privilege
// with no actions set at all (actions is Optional).
func TestNormalizePrivilegeSetHandlesEmptyActions(t *testing.T) {
	raw := map[string]interface{}{
		"name": "test_role",
		"privilege": []interface{}{
			map[string]interface{}{
				"db":         "admin",
				"collection": "",
				"actions":    []interface{}{},
			},
		},
	}
	d := schema.TestResourceDataRaw(t, resourceDatabaseRole().Schema, raw)

	privilegeRaw := d.Get("privilege").(*schema.Set).List()
	normalized := normalizePrivilegeSet(privilegeRaw)

	var privileges []PrivilegeDto
	if err := mapstructure.Decode(normalized, &privileges); err != nil {
		t.Fatalf("mapstructure.Decode failed on empty-actions privilege: %s", err)
	}
	if len(privileges) != 1 || len(privileges[0].Actions) != 0 {
		t.Fatalf("privileges = %+v, want one privilege with zero actions", privileges)
	}
}
