package mongodb

import "testing"

func TestResourceDatabaseCollectionParseId(t *testing.T) {
	db, name, err := resourceDatabaseCollectionParseId("mydb/mycollection")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if db != "mydb" || name != "mycollection" {
		t.Fatalf("got db=%q name=%q, want db=mydb name=mycollection", db, name)
	}
}

func TestResourceDatabaseCollectionParseIdInvalid(t *testing.T) {
	if _, _, err := resourceDatabaseCollectionParseId("not-a-valid-id"); err == nil {
		t.Fatal("expected error for malformed ID, got nil")
	}
}

func TestResourceDatabaseIndexParseId(t *testing.T) {
	db, collection, name, err := resourceDatabaseIndexParseId("mydb/mycollection/email_1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if db != "mydb" || collection != "mycollection" || name != "email_1" {
		t.Fatalf("got db=%q collection=%q name=%q, want mydb/mycollection/email_1", db, collection, name)
	}
}

func TestResourceDatabaseIndexParseIdInvalid(t *testing.T) {
	if _, _, _, err := resourceDatabaseIndexParseId("mydb/mycollection"); err == nil {
		t.Fatal("expected error for a two-part ID missing the index name, got nil")
	}
}

func TestDefaultIndexName(t *testing.T) {
	got := defaultIndexName([]indexKey{{Field: "email", Value: "1"}, {Field: "created_at", Value: "-1"}})
	want := "email_1_created_at_-1"
	if got != want {
		t.Fatalf("defaultIndexName() = %q, want %q", got, want)
	}
}

func TestIndexKeyValue(t *testing.T) {
	if v := indexKeyValue("1"); v != 1 {
		t.Errorf(`indexKeyValue("1") = %v (%T), want int 1`, v, v)
	}
	if v := indexKeyValue("-1"); v != -1 {
		t.Errorf(`indexKeyValue("-1") = %v (%T), want int -1`, v, v)
	}
	if v := indexKeyValue("text"); v != "text" {
		t.Errorf(`indexKeyValue("text") = %v, want string "text"`, v)
	}
	if v := indexKeyValue("2dsphere"); v != "2dsphere" {
		t.Errorf(`indexKeyValue("2dsphere") = %v, want string "2dsphere"`, v)
	}
}

func TestDecodeIndexKeysRejectsEmpty(t *testing.T) {
	if _, err := decodeIndexKeys(nil); err == nil {
		t.Fatal("expected error for zero keys, got nil")
	}
}
