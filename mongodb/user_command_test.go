package mongodb

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// commandKeys returns the set of top-level keys present in a built command, for asserting
// presence/absence without depending on bson.D's field order.
func commandKeys(cmd bson.D) map[string]bool {
	keys := make(map[string]bool, len(cmd))
	for _, e := range cmd {
		keys[e.Key] = true
	}
	return keys
}

func commandValue(t *testing.T, cmd bson.D, key string) interface{} {
	t.Helper()
	for _, e := range cmd {
		if e.Key == key {
			return e.Value
		}
	}
	t.Fatalf("command has no %q key: %+v", key, cmd)
	return nil
}

func TestBuildUserCommandPasswordAuthOmitsMechanisms(t *testing.T) {
	user := DbUser{Name: "alice", Password: "s3cret"}
	roles := []Role{{Role: "readWrite", Db: "admin"}}

	cmd := buildUserCommand("createUser", user, roles, nil)
	keys := commandKeys(cmd)

	if !keys["pwd"] {
		t.Error(`expected "pwd" key for a password-based user, got none`)
	}
	if keys["mechanisms"] {
		t.Error(`expected no "mechanisms" key when auth_mechanisms is unset`)
	}
	if commandValue(t, cmd, "createUser") != "alice" {
		t.Errorf(`expected createUser value "alice", got %v`, commandValue(t, cmd, "createUser"))
	}
	if commandValue(t, cmd, "pwd") != "s3cret" {
		t.Errorf(`expected pwd "s3cret", got %v`, commandValue(t, cmd, "pwd"))
	}
}

// This is the exact bug issue #42 reports: MongoDB/DocumentDB reject createUser/updateUser
// for an external-identity mechanism (e.g. IAM auth on DocumentDB) if a pwd field is present
// at all, even empty - so it must be omitted entirely, not just sent as "".
func TestBuildUserCommandIAMAuthOmitsPassword(t *testing.T) {
	user := DbUser{Name: "arn:aws:iam::123456789123:role/iamrole", Password: ""}
	roles := []Role{{Role: "readWrite", Db: "readWriteDb"}}

	cmd := buildUserCommand("createUser", user, roles, []string{"MONGODB-AWS"})
	keys := commandKeys(cmd)

	if keys["pwd"] {
		t.Error(`expected no "pwd" key for an IAM-authenticated (passwordless) user`)
	}
	if !keys["mechanisms"] {
		t.Error(`expected a "mechanisms" key when auth_mechanisms is set`)
	}
	mechanisms, ok := commandValue(t, cmd, "mechanisms").([]string)
	if !ok || len(mechanisms) != 1 || mechanisms[0] != "MONGODB-AWS" {
		t.Errorf(`expected mechanisms ["MONGODB-AWS"], got %v`, commandValue(t, cmd, "mechanisms"))
	}
}

func TestBuildUserCommandUpdateVerbAndEmptyRolesFallback(t *testing.T) {
	user := DbUser{Name: "bob", Password: "hunter2"}

	cmd := buildUserCommand("updateUser", user, nil, nil)
	keys := commandKeys(cmd)

	if commandValue(t, cmd, "updateUser") != "bob" {
		t.Errorf(`expected updateUser value "bob", got %v`, commandValue(t, cmd, "updateUser"))
	}
	if !keys["roles"] {
		t.Error(`expected a "roles" key even with no roles (empty-array fallback)`)
	}
	roles, ok := commandValue(t, cmd, "roles").([]bson.M)
	if !ok || len(roles) != 0 {
		t.Errorf(`expected an empty []bson.M roles fallback, got %v`, commandValue(t, cmd, "roles"))
	}
}

func TestBuildUserCommandNoPasswordNoMechanismsOmitsBoth(t *testing.T) {
	user := DbUser{Name: "carol", Password: ""}

	cmd := buildUserCommand("createUser", user, nil, nil)
	keys := commandKeys(cmd)

	if keys["pwd"] {
		t.Error(`expected no "pwd" key when password is unset`)
	}
	if keys["mechanisms"] {
		t.Error(`expected no "mechanisms" key when auth_mechanisms is unset`)
	}
}
