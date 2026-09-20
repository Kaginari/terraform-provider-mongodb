package mongodb

import "testing"

func TestResourceSchemasInternalValidate(t *testing.T) {
	if err := resourceDatabaseUser().InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceDatabaseUser schema invalid: %s", err)
	}
	if err := resourceDatabaseRole().InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceDatabaseRole schema invalid: %s", err)
	}
}

// MongoDB has no command to rename a user/role or move it to a different auth database -
// each of those is a different identity. These fields must be ForceNew so Update never
// silently targets the wrong user/role via updateUser/updateRole.
func TestResourceDatabaseUserIdentityFieldsAreForceNew(t *testing.T) {
	schema := resourceDatabaseUser().Schema
	if !schema["name"].ForceNew {
		t.Error(`"name" must be ForceNew`)
	}
	if !schema["auth_database"].ForceNew {
		t.Error(`"auth_database" must be ForceNew`)
	}
	if schema["password"].ForceNew {
		t.Error(`"password" must stay updatable in place`)
	}
}

func TestResourceDatabaseRoleIdentityFieldsAreForceNew(t *testing.T) {
	schema := resourceDatabaseRole().Schema
	if !schema["name"].ForceNew {
		t.Error(`"name" must be ForceNew`)
	}
	if !schema["database"].ForceNew {
		t.Error(`"database" must be ForceNew`)
	}
	if schema["privilege"].ForceNew {
		t.Error(`"privilege" must stay updatable in place`)
	}
}
