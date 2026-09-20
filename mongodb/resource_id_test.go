package mongodb

import (
	"context"
	"encoding/base64"
	"testing"
)

func TestResourceDatabaseUserParseId(t *testing.T) {
	userName, database, err := resourceDatabaseUserParseId("mydb/myuser")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if database != "mydb" || userName != "myuser" {
		t.Fatalf("got database=%q userName=%q, want database=mydb userName=myuser", database, userName)
	}
}

func TestResourceDatabaseUserParseIdUserNameWithSlash(t *testing.T) {
	userName, database, err := resourceDatabaseUserParseId("mydb/my/user")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if database != "mydb" || userName != "my/user" {
		t.Fatalf("got database=%q userName=%q, want database=mydb userName=my/user", database, userName)
	}
}

func TestResourceDatabaseUserParseIdInvalid(t *testing.T) {
	if _, _, err := resourceDatabaseUserParseId("not-a-valid-id"); err == nil {
		t.Fatal("expected error for malformed ID, got nil")
	}
	if _, _, err := resourceDatabaseUserParseId("/myuser"); err == nil {
		t.Fatal("expected error for empty database component, got nil")
	}
	if _, _, err := resourceDatabaseUserParseId("mydb/"); err == nil {
		t.Fatal("expected error for empty username component, got nil")
	}
}

func TestResourceDatabaseUserUpgradeV0(t *testing.T) {
	oldId := base64.StdEncoding.EncodeToString([]byte("mydb.myuser"))
	rawState := map[string]interface{}{"id": oldId}

	newState, err := resourceDatabaseUserUpgradeV0(context.Background(), rawState, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if newState["id"] != "mydb/myuser" {
		t.Fatalf("got id=%v, want mydb/myuser", newState["id"])
	}

	// Feeding an already-upgraded (plain) ID back through must be a no-op, not an error -
	// StateUpgraders must tolerate being invoked on state that's already in the new shape.
	alreadyNew := map[string]interface{}{"id": "mydb/myuser"}
	result, err := resourceDatabaseUserUpgradeV0(context.Background(), alreadyNew, nil)
	if err != nil {
		t.Fatalf("unexpected error on already-migrated state: %s", err)
	}
	if result["id"] != "mydb/myuser" {
		t.Fatalf("got id=%v, want unchanged mydb/myuser", result["id"])
	}
}

func TestResourceDatabaseRoleParseId(t *testing.T) {
	roleName, database, err := resourceDatabaseRoleParseId("admin/myrole")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if database != "admin" || roleName != "myrole" {
		t.Fatalf("got database=%q roleName=%q, want database=admin roleName=myrole", database, roleName)
	}
}

func TestResourceDatabaseRoleUpgradeV0(t *testing.T) {
	oldId := base64.StdEncoding.EncodeToString([]byte("admin.myrole"))
	rawState := map[string]interface{}{"id": oldId}

	newState, err := resourceDatabaseRoleUpgradeV0(context.Background(), rawState, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if newState["id"] != "admin/myrole" {
		t.Fatalf("got id=%v, want admin/myrole", newState["id"])
	}
}

func TestResourceDatabaseUserAndRoleSchemaVersionsHaveUpgraders(t *testing.T) {
	user := resourceDatabaseUser()
	if user.SchemaVersion != 1 {
		t.Fatalf("resourceDatabaseUser SchemaVersion = %d, want 1", user.SchemaVersion)
	}
	if len(user.StateUpgraders) != 1 || user.StateUpgraders[0].Version != 0 {
		t.Fatalf("resourceDatabaseUser missing v0->v1 StateUpgrader")
	}

	role := resourceDatabaseRole()
	if role.SchemaVersion != 1 {
		t.Fatalf("resourceDatabaseRole SchemaVersion = %d, want 1", role.SchemaVersion)
	}
	if len(role.StateUpgraders) != 1 || role.StateUpgraders[0].Version != 0 {
		t.Fatalf("resourceDatabaseRole missing v0->v1 StateUpgrader")
	}
}
