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

func TestResourceDatabaseCollectionSchemaInternalValidate(t *testing.T) {
	if err := resourceDatabaseCollection().InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceDatabaseCollection schema invalid: %s", err)
	}
}

func TestResourceDatabaseCollectionIdentityFieldsAreForceNew(t *testing.T) {
	schema := resourceDatabaseCollection().Schema
	if !schema["db"].ForceNew {
		t.Error(`"db" must be ForceNew`)
	}
	if !schema["name"].ForceNew {
		t.Error(`"name" must be ForceNew`)
	}
	if schema["change_stream_pre_and_post_images"].ForceNew {
		t.Error(`"change_stream_pre_and_post_images" must stay updatable in place (via collMod)`)
	}
}

func TestResourceDatabaseCollectionDeletionProtectionDefaultsTrue(t *testing.T) {
	schema := resourceDatabaseCollection().Schema
	if schema["deletion_protection"].Default != true {
		t.Error(`"deletion_protection" must default to true - deleting a collection is destructive`)
	}
}

func TestResourceDatabaseIndexSchemaInternalValidate(t *testing.T) {
	if err := resourceDatabaseIndex().InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceDatabaseIndex schema invalid: %s", err)
	}
}

func TestResourceDatabaseIndexIdentityFieldsAreForceNew(t *testing.T) {
	schema := resourceDatabaseIndex().Schema
	for _, field := range []string{"db", "collection", "name", "keys", "partial_filter_expression"} {
		if !schema[field].ForceNew {
			t.Errorf("%q must be ForceNew - changing it means a different index", field)
		}
	}
	if schema["hidden"].ForceNew {
		t.Error(`"hidden" must stay updatable in place (via collMod) - MongoDB doesn't require a rebuild to toggle it`)
	}
}
